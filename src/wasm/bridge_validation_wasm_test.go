//go:build js && wasm

package wasm

import (
	"bytes"
	"encoding/binary"
	"image"
	stdjpeg "image/jpeg"
	"math"
	"strings"
	"syscall/js"
	"testing"

	"github.com/mac/go-pixo/src/png"
)

func TestHandlersRejectMalformedArguments(t *testing.T) {
	bytes := js.Global().Get("Uint8Array").New(4)
	object := js.Global().Get("Object")
	bytePrototype := js.Global().Get("Uint8Array").Get("prototype")
	spoofedView := js.Global().Get("DataView").New(js.Global().Get("ArrayBuffer").New(4))
	object.Call("setPrototypeOf", spoofedView, bytePrototype)
	spoofedWords := js.Global().Get("Uint16Array").New(2)
	object.Call("setPrototypeOf", spoofedWords, bytePrototype)
	tests := []struct {
		name   string
		handle func(js.Value, []js.Value) any
		args   []any
	}{
		{"png object", HandleEncodePng, []any{map[string]any{"length": 4}, 1, 1, 6, 2, false, 0}},
		{"png spoofed DataView", HandleEncodePng, []any{spoofedView, 1, 1, 6, 2, false, 0}},
		{"png spoofed Uint16Array", HandleEncodePng, []any{spoofedWords, 1, 1, 6, 2, false, 0}},
		{"advanced spoofed DataView", HandleEncodePngAdvanced, []any{spoofedView, 1, 1, 6, 2, false, 0, false, 0.5, 75, 0}},
		{"jpeg spoofed DataView", HandleEncodeJpeg, []any{spoofedView, 4, 1, 1, 75}},
		{"jpeg advanced spoofed DataView", HandleEncodeJpegAdvanced, []any{spoofedView, 4, 1, 1, 75, 0, false, false, false, 1}},
		{"png recompress spoofed DataView", HandleRecompressPngLossless, []any{spoofedView, 2, 0}},
		{"jpeg recompress spoofed DataView", HandleRecompressJpeg, []any{spoofedView, 1, 75, false, false, false}},
		{"png truncated", HandleEncodePng, []any{bytes, 2, 1, 6, 2, false, 0}},
		{"png zero", HandleEncodePng, []any{bytes, 0, 1, 6, 2, false, 0}},
		{"png negative", HandleEncodePng, []any{bytes, -1, 1, 6, 2, false, 0}},
		{"png overflow", HandleEncodePng, []any{bytes, float64(1<<61) + 1, 8, 0, 2, false, 0}},
		{"png fractional", HandleEncodePng, []any{bytes, 1.5, 1, 6, 2, false, 0}},
		{"png NaN", HandleEncodePng, []any{bytes, math.NaN(), 1, 6, 2, false, 0}},
		{"png string", HandleEncodePng, []any{bytes, "1", 1, 6, 2, false, 0}},
		{"png boolean", HandleEncodePng, []any{bytes, 1, 1, 6, 2, 1, 0}},
		{"png preset", HandleEncodePng, []any{bytes, 1, 1, 6, -1, false, 0}},
		{"png fallback policy type", HandleEncodePng, []any{bytes, 1, 1, 6, 2, false, 0, "true"}},
		{"png colors", HandleEncodePng, []any{bytes, 1, 1, 6, 2, true, 257}},
		{"advanced strength", HandleEncodePngAdvanced, []any{bytes, 1, 1, 6, 2, true, 2, false, math.Inf(1), 75, 0}},
		{"advanced iterations", HandleEncodePngAdvanced, []any{bytes, 1, 1, 6, 2, false, 0, false, 0.5, 75, 100000}},
		{"advanced partial options", HandleEncodePngAdvanced, []any{bytes, 1, 1, 6, 2, false, 0, false, 0.5, 75, 0, 2}},
		{"advanced filter", HandleEncodePngAdvanced, []any{bytes, 1, 1, 6, 2, false, 0, false, 0.5, 75, 0, 999, false, 0, false, false}},
		{"advanced distance", HandleEncodePngAdvanced, []any{bytes, 1, 1, 6, 2, false, 0, false, 0.5, 75, 0, 0, false, 99, false, false}},
		{"advanced fallback policy type", HandleEncodePngAdvanced, []any{bytes, 1, 1, 6, 2, false, 0, false, 0.5, 75, 0, 0, false, 0, false, false, js.Undefined(), "true"}},
		{"jpeg wrapped quality", HandleEncodeJpeg, []any{bytes.Call("subarray", 0, 3), 1, 1, 3, 331}},
		{"jpeg short", HandleEncodeJpeg, []any{js.Global().Get("Uint8Array").New(2), 1, 1, 3, 75}},
		{"jpeg subsampling", HandleEncodeJpegAdvanced, []any{bytes, 1, 1, 3, 75, 99, false, false, false, 1}},
		{"jpeg recompress type", HandleRecompressJpeg, []any{js.Null(), 1, 75, false, false, false}},
		{"png recompress callback", HandleRecompressPngLossless, []any{bytes, 2, 0, 42}},
		{"bpp type", HandleBytesPerPixel, []any{"6"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("handler panicked: %v", p)
				}
			}()
			args := make([]js.Value, len(tt.args))
			for i, a := range tt.args {
				args[i] = js.ValueOf(a)
			}
			result := tt.handle(js.Undefined(), args)
			value, ok := result.(js.Value)
			if !ok || value.Type() != js.TypeString || !strings.Contains(value.String(), "error:") {
				t.Errorf("want error string, got %v", result)
			}
		})
	}
}

func TestHandlersRejectTypedArrayInspectionExceptions(t *testing.T) {
	pixels := js.Global().Get("Uint8Array").New(4)
	object := js.Global().Get("Object")
	prototype := object.Call("getPrototypeOf", js.Global().Get("Uint8Array").Get("prototype"))
	descriptor := object.Call("getOwnPropertyDescriptor", prototype, "byteLength")
	defer object.Call("defineProperty", prototype, "byteLength", descriptor)
	object.Call("defineProperty", prototype, "byteLength", map[string]any{
		"get": js.Global().Get("Function").New("throw new TypeError('inspection failed')"), "configurable": true,
	})
	defer func() {
		if p := recover(); p != nil {
			t.Errorf("JS inspection exception escaped the handler: %v", p)
		}
	}()
	result := HandleEncodePng(js.Undefined(), []js.Value{pixels, js.ValueOf(1), js.ValueOf(1), js.ValueOf(6), js.ValueOf(2), js.ValueOf(false), js.ValueOf(0)}).(js.Value)
	if result.Type() != js.TypeString || !strings.Contains(result.String(), "error:") {
		t.Errorf("want error string, got %v", result)
	}
}

func TestRecompressHandlersRejectOversizedHeaders(t *testing.T) {
	var p bytes.Buffer
	p.Write(png.Signature())
	ihdr, err := png.NewIHDRData(5000, 5000, 8, 6)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.WriteIHDR(&p, ihdr); err != nil {
		t.Fatal(err)
	}
	var j bytes.Buffer
	if err := stdjpeg.Encode(&j, image.NewGray(image.Rect(0, 0, 1, 1)), nil); err != nil {
		t.Fatal(err)
	}
	jfif := []byte{0xff, 0xd8, 0xff, 0xe0, 0, 16, 'J', 'F', 'I', 'F', 0, 1, 1, 0, 0, 1, 0, 1, 0, 0}
	header := make([]byte, 0, len(jfif)+j.Len()-2)
	header = append(header, jfif...)
	header = append(header, j.Bytes()[2:]...)
	sof := bytes.Index(header, []byte{0xff, 0xc0})
	if sof < 0 {
		t.Fatal("missing SOF")
	}
	binary.BigEndian.PutUint16(header[sof+5:], 5000)
	binary.BigEndian.PutUint16(header[sof+7:], 5000)
	header = header[:sof+2+int(binary.BigEndian.Uint16(header[sof+2:]))]
	for _, tt := range []struct {
		name    string
		input   []byte
		handle  func(js.Value, []js.Value) any
		options []any
	}{
		{"png", p.Bytes(), HandleRecompressPngLossless, []any{2, 0}},
		{"jpeg", header, HandleRecompressJpeg, []any{1, 75, false, false, false}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := js.Global().Get("Uint8Array").New(len(tt.input))
			js.CopyBytesToJS(input, tt.input)
			args := []js.Value{input}
			for _, value := range tt.options {
				args = append(args, js.ValueOf(value))
			}
			result := tt.handle(js.Undefined(), args).(js.Value)
			if result.Type() != js.TypeString || !strings.Contains(result.String(), "limit") {
				t.Fatalf("want resource limit before full decode, got %v", result)
			}
		})
	}
}

func TestHandlersKeepOrdinaryCallsWorking(t *testing.T) {
	pixels := js.Global().Get("Uint8Array").New(4)
	clamped := js.Global().Get("Uint8ClampedArray").New(4)
	opaque := js.Global().Get("Uint8Array").New(4)
	js.CopyBytesToJS(opaque, []byte{12, 34, 56, 255})
	view := js.Global().Get("Uint8Array").New(8).Call("subarray", 4, 8)
	viewBytes := []byte{12, 34, 56, 255}
	js.CopyBytesToJS(view, viewBytes)
	hostileView := js.Global().Get("Uint8Array").New(4)
	js.CopyBytesToJS(hostileView, viewBytes)
	prototypeProxy := js.Global().Get("Proxy").New(js.Global().Get("Object").Call("create", js.Null()), map[string]any{
		"getPrototypeOf": js.Global().Get("Function").New("throw new TypeError('untrusted prototype inspection')"),
	})
	js.Global().Get("Object").Call("setPrototypeOf", hostileView, prototypeProxy)
	js.Global().Get("Object").Call("defineProperty", view, "subarray", map[string]any{
		"value": js.Global().Get("Function").New("throw new TypeError('untrusted subarray called')"),
	})
	if copied, err := copyJSBytes(view); err != nil || !bytes.Equal(copied, viewBytes) {
		t.Fatalf("subview bytes/offset not preserved: %v, %v", copied, err)
	}
	// Own branding can also lie; validation must inspect the internal view type.
	js.Global().Get("Object").Call("defineProperty", pixels, js.Global().Get("Symbol").Get("toStringTag"), map[string]any{"value": "DataView"})
	// A misleading own length must not drive allocation or cause truncation.
	js.Global().Get("Object").Call("defineProperty", clamped, "length", map[string]any{"value": 1024})
	callback := js.FuncOf(func(js.Value, []js.Value) any { return nil })
	defer callback.Release()
	for _, tt := range []struct {
		name   string
		handle func(js.Value, []js.Value) any
		args   []any
	}{
		{"png", HandleEncodePng, []any{pixels, 1, 1, 6, 2, false, 0}},
		{"png preset RGBA fallback", HandleEncodePng, []any{pixels, 1, 1, 6, 2, true, 256, true}},
		{"png one-color palette", HandleEncodePng, []any{opaque, 1, 1, 6, 2, true, 1}},
		{"png intrinsic subview copy", HandleEncodePng, []any{view, 1, 1, 6, 2, false, 0}},
		{"png hostile view prototype", HandleEncodePng, []any{hostileView, 1, 1, 6, 2, false, 0}},
		{"png clamped real size", HandleEncodePng, []any{clamped, 1, 1, 6, 2, false, 0}},
		{"png advanced legacy", HandleEncodePngAdvanced, []any{pixels, 1, 1, 6, 2, false, 0, false, 0.5, 75, 0}},
		{"png advanced legacy callback", HandleEncodePngAdvanced, []any{pixels, 1, 1, 6, 2, false, 0, false, 0.5, 75, 0, callback.Value}},
		{"png advanced full callback", HandleEncodePngAdvanced, []any{pixels, 1, 1, 6, 2, false, 0, false, 0.5, 75, 0, 0, false, 0, false, false, callback.Value}},
		{"png advanced preset RGBA fallback", HandleEncodePngAdvanced, []any{pixels, 1, 1, 6, 2, true, 256, false, 0.5, 75, 0, 0, false, 0, false, false, callback.Value, true}},
		{"png advanced one-color palette", HandleEncodePngAdvanced, []any{opaque, 1, 1, 6, 2, true, 1, false, 0.5, 75, 0}},
		{"jpeg", HandleEncodeJpeg, []any{pixels.Call("subarray", 0, 3), 1, 1, 3, 75}},
		{"jpeg advanced", HandleEncodeJpegAdvanced, []any{pixels.Call("subarray", 0, 3), 1, 1, 3, 75, 0, false, false, true, 1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := make([]js.Value, len(tt.args))
			for i, a := range tt.args {
				args[i] = js.ValueOf(a)
			}
			result := tt.handle(js.Undefined(), args).(js.Value)
			if !result.InstanceOf(js.Global().Get("Uint8Array")) || result.Get("length").Int() == 0 {
				t.Fatalf("want encoded bytes, got %v", result)
			}
		})
	}
}
