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
	js.Global().Set("encodePngAdvanced", js.FuncOf(wasm.HandleEncodePngAdvanced))
	js.Global().Set("encodeJpeg", js.FuncOf(wasm.HandleEncodeJpeg))
	js.Global().Set("encodeJpegAdvanced", js.FuncOf(wasm.HandleEncodeJpegAdvanced))
	js.Global().Set("recompressPngLossless", js.FuncOf(wasm.HandleRecompressPngLossless))
	js.Global().Set("bytesPerPixel", js.FuncOf(wasm.HandleBytesPerPixel))

	// Signal that the WASM is ready
	if initFunc := js.Global().Get("goWasmInit"); initFunc.Truthy() {
		initFunc.Invoke()
	}

	<-c
}
