//go:build !linux

package knob

import "context"

func LocalConsoleActive() bool { return false }

func Watch(context.Context, string, func() bool) <-chan Event { return make(chan Event) }
