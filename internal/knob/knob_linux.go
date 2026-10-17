//go:build linux

package knob

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// LocalConsoleActive excludes SSH, pseudo-terminals and inactive virtual consoles.
func LocalConsoleActive() bool {
	dev, err := unix.IoctlGetInt(int(os.Stdin.Fd()), unix.TIOCGDEV)
	if err != nil || unix.Major(uint64(dev)) != 4 {
		return false
	}
	n := unix.Minor(uint64(dev))
	if n < 1 || n > 63 {
		return false
	}
	active, err := os.ReadFile("/sys/class/tty/tty0/active")
	return err == nil && strings.TrimSpace(string(active)) == fmt.Sprintf("tty%d", n)
}

// Watch reconnects after hotplug. Permission failures or a missing optional knob
// leave normal terminal input working. No grab is taken on the typing device.
func Watch(ctx context.Context, path string, active func() bool) <-chan Event {
	out := make(chan Event)
	if path == "" {
		return out
	}
	go func() {
		fd := -1
		defer func() {
			if fd >= 0 {
				unix.Close(fd)
			}
		}()
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		var retry time.Time
		size := int(unsafe.Sizeof(unix.Timeval{})) + 8
		buffer := make([]byte, size*64)
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if fd < 0 {
					if now.Before(retry) {
						continue
					}
					retry = now.Add(time.Second)
					fd, _ = unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
					if fd < 0 {
						continue
					}
				}
				n, err := unix.Read(fd, buffer)
				if err == unix.EAGAIN || err == unix.EINTR {
					continue
				}
				if err != nil || n == 0 {
					unix.Close(fd)
					fd = -1
					continue
				}
				foreground := active()
				for offset := 0; offset+size <= n; offset += size {
					tail := buffer[offset+size-8 : offset+size]
					event, ok := decode(binary.NativeEndian.Uint16(tail), binary.NativeEndian.Uint16(tail[2:]), int32(binary.NativeEndian.Uint32(tail[4:])))
					if ok && foreground {
						select {
						case out <- event:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()
	return out
}
