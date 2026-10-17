//go:build linux

package knob

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

func TestWatchReconnectsAndIgnoresInactiveConsole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "knob")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var active atomic.Bool
	events := Watch(ctx, path, active.Load)
	// Start without a device, then connect it.
	if err := unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { unix.Close(fd) }()
	send := func(code uint16, value uint32) {
		t.Helper()
		data := make([]byte, int(unsafe.Sizeof(unix.Timeval{}))+8)
		tail := data[len(data)-8:]
		binary.NativeEndian.PutUint16(tail, 1)
		binary.NativeEndian.PutUint16(tail[2:], code)
		binary.NativeEndian.PutUint32(tail[4:], value)
		if _, err := unix.Write(fd, data); err != nil {
			t.Fatal(err)
		}
	}
	send(115, 1)
	// Wait until the inactive event has actually been drained, including the
	// case where Watch first saw a missing device and is retrying the open.
	deadline := time.Now().Add(2 * time.Second)
	for {
		pending, err := unix.IoctlGetInt(fd, unix.TIOCINQ)
		if err != nil {
			t.Fatal(err)
		}
		if pending == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("inactive event was not drained")
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case event := <-events:
		t.Fatalf("inactive console received %+v", event)
	default:
	}
	active.Store(true)
	send(113, 2) // holding the button cannot repeatedly toggle the target
	send(114, 1)
	select {
	case event := <-events:
		if event != (Event{Step: -1}) {
			t.Fatalf("got %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("hotplugged device did not deliver media event")
	}
	// Replace the node to model unplug/replug with a new evdev device.
	unix.Close(fd)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	fd, err = unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		t.Fatal(err)
	}
	send(115, 1)
	select {
	case event := <-events:
		if event != (Event{Step: 1}) {
			t.Fatalf("after reconnect: %+v", event)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("replacement device did not reconnect")
	}
}
