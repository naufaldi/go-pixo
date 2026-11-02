//go:build js && wasm

package wasm

import "syscall/js"

// HandleValidateImageBytes validates encoded bytes before accepting an unchanged result.
func HandleValidateImageBytes(_ js.Value, args []js.Value) any {
	if err := checkArguments(args, nil); err != nil {
		return argumentError(err)
	}
	if err := checkEncodedBytes(args[0]); err != nil {
		return argumentError(err)
	}
	input, err := copyJSBytes(args[0])
	if err != nil {
		return argumentError(err)
	}
	if err := ValidateImageBytes(input); err != nil {
		return argumentError(err)
	}
	return js.ValueOf(true)
}
