package typeutil

import (
	"fmt"
	"runtime"
)

func StackTrace() []string {
	buf := make([]uintptr, 32)
	n := runtime.Callers(2, buf) // 2 for ignoring StackTrace and runtime.Callers
	frames := runtime.CallersFrames(buf[:n])

	var stack []string
	for {
		frame, more := frames.Next()
		stack = append(stack, fmt.Sprintf("%s:%d", frame.File, frame.Line))
		if !more {
			break
		}
	}
	return stack
}
