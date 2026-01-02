package compress

// DeflateError represents errors for DEFLATE encoding operations.
type DeflateError string

func (e DeflateError) Error() string {
	return string(e)
}

const (
	ErrInvalidSymbol   DeflateError = "invalid symbol"
	ErrInvalidLength   DeflateError = "invalid length"
	ErrInvalidDistance DeflateError = "invalid distance"
	ErrInvalidHLIT     DeflateError = "invalid HLIT"
	ErrInvalidHDIST    DeflateError = "invalid HDIST"
	ErrInvalidHCLEN    DeflateError = "invalid HCLEN"
)

// EncodeLiteral writes a literal symbol (0-255) or end-of-block (256) to the bit writer.
func EncodeLiteral(w *BitWriter, symbol int, table Table) error {
	if symbol < 0 {
		return ErrInvalidSymbol
	}
	if symbol >= len(table.Codes) {
		return ErrInvalidSymbol
	}
	
	code := table.Codes[symbol]
	if code.Length == 0 {
		return ErrInvalidSymbol
	}
	
	return w.Write(code.Bits, code.Length)
}

// EncodeLength writes a match length (3-258) to the bit writer.
// The length is encoded as a code (257-285) plus optional extra bits.
func EncodeLength(w *BitWriter, length int, table Table) error {
	if length < MinMatchLength || length > MaxMatchLength {
		return ErrInvalidLength
	}
	
	code := findLengthCode(length)
	if code < 257 || code > 285 {
		return ErrInvalidLength
	}
	
	if code-257 >= len(table.Codes) {
		return ErrInvalidSymbol
	}
	
	huffmanCode := table.Codes[code]
	if huffmanCode.Length == 0 {
		return ErrInvalidSymbol
	}
	
	if err := w.Write(huffmanCode.Bits, huffmanCode.Length); err != nil {
		return err
	}
	
	extraBits := LengthExtraBits[code-257]
	if extraBits > 0 {
		base := LengthBase[code-257]
		extraValue := uint16(length - int(base))
		return w.Write(extraValue, int(extraBits))
	}
	
	return nil
}

// EncodeDistance writes a distance (1-32768) to the bit writer.
// The distance is encoded as a code (0-29) plus optional extra bits.
func EncodeDistance(w *BitWriter, distance int, table Table) error {
	if distance < 1 || distance > MaxDistance {
		return ErrInvalidDistance
	}
	
	code := findDistanceCode(distance)
	if code < 0 || code >= 30 {
		return ErrInvalidDistance
	}
	
	if code >= len(table.Codes) {
		return ErrInvalidSymbol
	}
	
	huffmanCode := table.Codes[code]
	if huffmanCode.Length == 0 {
		return ErrInvalidSymbol
	}
	
	if err := w.Write(huffmanCode.Bits, huffmanCode.Length); err != nil {
		return err
	}
	
	extraBits := DistanceExtraBits[code]
	if extraBits > 0 {
		base := DistanceBase[code]
		extraValue := uint16(distance - int(base))
		return w.Write(extraValue, int(extraBits))
	}
	
	return nil
}

// findLengthCode finds the length code (257-285) for a given length (3-258).
func findLengthCode(length int) int {
	for code := 0; code < len(LengthBase); code++ {
		base := int(LengthBase[code])
		extraBits := LengthExtraBits[code]
		maxLength := base + (1 << extraBits) - 1
		
		if length >= base && length <= maxLength {
			return 257 + code
		}
	}
	return -1
}

// findDistanceCode finds the distance code (0-29) for a given distance (1-32768).
func findDistanceCode(distance int) int {
	for code := 0; code < len(DistanceBase); code++ {
		base := int(DistanceBase[code])
		extraBits := DistanceExtraBits[code]
		maxDistance := base + (1 << extraBits) - 1
		
		if distance >= base && distance <= maxDistance {
			return code
		}
	}
	return -1
}
