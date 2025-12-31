//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/mac/go-pixo/src/wasm"
)

func main() {
	c := make(chan struct{}, 0)
	
	// Register functions
	js.Global().Set("encodePng", js.FuncOf(wasm.HandleEncodePng))
	js.Global().Set("bytesPerPixel", js.FuncOf(wasm.HandleBytesPerPixel))
	
	// Signal that the WASM is ready
	if initFunc := js.Global().Get("goWasmInit"); initFunc.Truthy() {
		initFunc.Invoke()
	}

	<-c
}
