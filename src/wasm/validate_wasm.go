//go:build js && wasm

package wasm

import (
	"fmt"
	"math"
	"syscall/js"

	"github.com/mac/go-pixo/src/internal/imagelimit"
	"github.com/mac/go-pixo/src/png"
)

type argumentRule struct {
	kind     js.Type
	min, max float64
	integer  bool
}

func integerArg(min, max int) argumentRule {
	return argumentRule{kind: js.TypeNumber, min: float64(min), max: float64(max), integer: true}
}

func numberArg(min, max float64) argumentRule {
	return argumentRule{kind: js.TypeNumber, min: min, max: max}
}

var booleanArg = argumentRule{kind: js.TypeBoolean}

// checkArguments runs before Int/Bool conversions or any input buffer copies.
// The first argument is a byte array, checked separately using its intrinsic size.
func checkArguments(args []js.Value, rules []argumentRule) error {
	if len(args) < len(rules)+1 {
		return fmt.Errorf("invalid arguments: expected at least %d arguments", len(rules)+1)
	}
	for i, rule := range rules {
		v := args[i+1]
		if v.Type() != rule.kind {
			return fmt.Errorf("argument %d must be %s", i+2, rule.kind)
		}
		if rule.kind == js.TypeNumber {
			n := v.Float()
			if math.IsNaN(n) || math.IsInf(n, 0) || n < rule.min || n > rule.max || (rule.integer && math.Trunc(n) != n) {
				kind := "number"
				if rule.integer {
					kind = "integer"
				}
				return fmt.Errorf("argument %d must be a finite %s in [%g, %g]", i+2, kind, rule.min, rule.max)
			}
		}
	}
	return nil
}

func byteArrayLength(value js.Value) (length int, err error) {
	// Foreign JS inspection can throw, including on prototype-spoofed views.
	// Keep that exception inside the argument boundary, before allocation/copy.
	defer func() {
		if recover() != nil {
			length = 0
			err = fmt.Errorf("could not inspect input byte array")
		}
	}()
	if value.Type() != js.TypeObject || !js.Global().Get("ArrayBuffer").Call("isView", value).Bool() {
		return 0, fmt.Errorf("input must be a Uint8Array or Uint8ClampedArray")
	}
	// Do not trust an own 'length' property shadowing the typed array's real size.
	object := js.Global().Get("Object")
	prototype := object.Call("getPrototypeOf", js.Global().Get("Uint8Array").Get("prototype"))
	// Never use instanceof on caller input: a prototype proxy can throw outside
	// Go's recover boundary. This getter reads the internal typed-array kind,
	// ignoring prototypes and an own Symbol.toStringTag property.
	tagGetter := object.Call("getOwnPropertyDescriptor", prototype, js.Global().Get("Symbol").Get("toStringTag")).Get("get")
	kind := tagGetter.Call("call", value)
	if kind.Type() != js.TypeString || (kind.String() != "Uint8Array" && kind.String() != "Uint8ClampedArray") {
		return 0, fmt.Errorf("input must be a Uint8Array or Uint8ClampedArray")
	}
	getter := object.Call("getOwnPropertyDescriptor", prototype, "byteLength").Get("get")
	return getter.Call("call", value).Int(), nil
}

func checkRawPixels(args []js.Value, isPNG bool) error {
	colorType := args[3].Int()
	bpp := colorType
	if isPNG {
		switch colorType {
		case 0:
			bpp = 1
		case 2:
			bpp = 3
		case 6:
			bpp = 4
		default:
			return fmt.Errorf("unsupported PNG color type: %d", colorType)
		}
	} else if colorType != 1 && colorType != 3 {
		return fmt.Errorf("unsupported JPEG color type: %d", colorType)
	}
	expected, err := imagelimit.PixelBytes(args[1].Int(), args[2].Int(), bpp)
	if err != nil {
		return err
	}
	actual, err := byteArrayLength(args[0])
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("pixel data length %d does not match expected %d", actual, expected)
	}
	return nil
}

func checkEncodedBytes(value js.Value) error {
	n, err := byteArrayLength(value)
	if err != nil {
		return err
	}
	return imagelimit.CheckEncodedSize(n)
}

func copyJSBytes(value js.Value) (data []byte, err error) {
	defer func() {
		if recover() != nil {
			data = nil
			err = fmt.Errorf("could not prepare input byte array")
		}
	}()
	n, err := byteArrayLength(value)
	if err != nil {
		return nil, err
	}
	// Callers enforce either exact raster length or the tighter encoded-file cap.
	if n <= 0 || n > imagelimit.MaxPixels*4 {
		return nil, fmt.Errorf("input buffer exceeds limit")
	}
	// Go's copy helper invokes src.subarray. Give it an owned, canonical view
	// rather than invoking methods or shadowed properties on the caller's view.
	// This shares the existing bytes and preserves the original view offset.
	object := js.Global().Get("Object")
	prototype := object.Call("getPrototypeOf", js.Global().Get("Uint8Array").Get("prototype"))
	buffer := object.Call("getOwnPropertyDescriptor", prototype, "buffer").Get("get").Call("call", value)
	offset := object.Call("getOwnPropertyDescriptor", prototype, "byteOffset").Get("get").Call("call", value)
	view := js.Global().Get("Uint8Array").New(buffer, offset, n)
	data = make([]byte, n)
	if js.CopyBytesToGo(data, view) != n {
		return nil, fmt.Errorf("could not copy complete input buffer")
	}
	return data, nil
}

func checkCallback(args []js.Value, index int) error {
	if len(args) > index && !args[index].IsUndefined() && !args[index].IsNull() && args[index].Type() != js.TypeFunction {
		return fmt.Errorf("argument %d must be a progress callback", index+1)
	}
	return nil
}

func checkPNGArguments(args []js.Value, advanced bool) error {
	rules := []argumentRule{integerArg(1, imagelimit.MaxDimension), integerArg(1, imagelimit.MaxDimension), integerArg(0, 6), integerArg(0, 4), booleanArg, integerArg(0, 256)}
	if advanced {
		rules = append(rules, booleanArg, numberArg(0, 1), integerArg(0, 100), integerArg(0, 50))
	}
	if err := checkArguments(args, rules); err != nil {
		return err
	}
	if args[5].Bool() && args[6].Int() < 2 {
		return fmt.Errorf("lossy PNG requires 2 to 256 colors")
	}
	if advanced {
		switch {
		case len(args) <= 12:
			if err := checkCallback(args, 11); err != nil {
				return err
			}
		case len(args) >= 16:
			rules = append(rules, integerArg(0, int(png.FilterStrategyCombined)), booleanArg, integerArg(0, int(png.DistanceMetricRedmean)), booleanArg, booleanArg)
			if err := checkArguments(args, rules); err != nil {
				return err
			}
			if err := checkCallback(args, 16); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid arguments: incomplete advanced PNG options")
		}
	}
	return checkRawPixels(args, true)
}

func checkJPEGArguments(args []js.Value, advanced bool) error {
	rules := []argumentRule{integerArg(1, imagelimit.MaxDimension), integerArg(1, imagelimit.MaxDimension), integerArg(1, 3), integerArg(1, 100)}
	if advanced {
		rules = append(rules, integerArg(0, 1), booleanArg, booleanArg, booleanArg, integerArg(0, 4))
	}
	if err := checkArguments(args, rules); err != nil {
		return err
	}
	return checkRawPixels(args, false)
}

func argumentError(err error) js.Value {
	return js.ValueOf(fmt.Sprintf("error: %v", err))
}
