package vm

import "time"

var hostInstructions uint32

// Go/WASM shares the browser's main thread. A tight Python loop must hand back
// control occasionally so the editor can process Run/Stop and remove its frame.
func yieldToHost() {
	hostInstructions++
	if hostInstructions&4095 == 0 {
		time.Sleep(time.Millisecond)
	}
}
