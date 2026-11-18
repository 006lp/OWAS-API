package protostuff

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"unicode/utf8"
)

// Protobuf / Protostuff wire types matching io.protostuff.C1752c and C1761l.
const (
	WireVarint     uint8 = 0
	Wire64Bit      uint8 = 1
	WireLen        uint8 = 2
	WireStartGroup uint8 = 3
	WireEndGroup   uint8 = 4
	Wire32Bit      uint8 = 5
)

// PolyClassTag carries the Java FQCN in Protostuff runtime polymorphic POJOs
// (morph_non_final_pojos=true).
const PolyClassTag uint64 = 127

const maxGroupDepth = 32

// RawField represents a single parsed wire-level field.
type RawField struct {
	Tag      uint64
	WireType uint8
	Varint   uint64
	Bytes    []byte
	Group    []RawField
}

// Int32 interprets a varint field as a signed 32-bit integer.
func (f RawField) Int32() int32 {
	return int32(f.Varint)
}

// Int64 interprets a varint field as a signed 64-bit integer.
func (f RawField) Int64() int64 {
	return int64(f.Varint)
}

// Bool interprets a varint field as a boolean.
func (f RawField) Bool() bool {
	return f.Varint != 0
}

// String interprets a length-delimited field as a UTF-8 string.
func (f RawField) String() string {
	return string(f.Bytes)
}

// Float32 decodes a 32-bit IEEE 754 little-endian float.
func (f RawField) Float32() (float32, error) {
	if len(f.Bytes) != 4 {
		return 0, fmt.Errorf("expected 4 bytes for float32, got %d", len(f.Bytes))
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(f.Bytes)), nil
}

// Float64 decodes a 64-bit IEEE 754 little-endian float.
func (f RawField) Float64() (float64, error) {
	if len(f.Bytes) != 8 {
		return 0, fmt.Errorf("expected 8 bytes for float64, got %d", len(f.Bytes))
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(f.Bytes)), nil
}

// PreviewValue returns a JSON-friendly representation for diagnostics/unknown fields.
func (f RawField) PreviewValue() any {
	switch f.WireType {
	case WireLen:
		if utf8.Valid(f.Bytes) {
			return string(f.Bytes)
		}
		return hex.EncodeToString(f.Bytes)
	case Wire32Bit, Wire64Bit:
		return hex.EncodeToString(f.Bytes)
	case WireStartGroup:
		items := make([]map[string]any, 0, len(f.Group))
		for _, g := range f.Group {
			items = append(items, map[string]any{
				"tag":       g.Tag,
				"wire_type": g.WireType,
				"value":     g.PreviewValue(),
			})
		}
		return items
	default:
		return f.Varint
	}
}

// AppendVarint appends a standard base-128 unsigned varint to dst.
func AppendVarint(dst []byte, v uint64) []byte {
	for v > 0x7F {
		dst = append(dst, byte(v&0x7F)|0x80)
		v >>= 7
	}
	return append(dst, byte(v))
}

// DecodeVarint decodes a base-128 varint from buf starting at pos.
func DecodeVarint(buf []byte, pos int) (uint64, int, error) {
	var result uint64
	var shift uint
	for pos < len(buf) {
		b := buf[pos]
		pos++
		result |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			return result, pos, nil
		}
		shift += 7
		if shift >= 70 {
			return 0, pos, errors.New("malformed varint (>10 bytes)")
		}
	}
	return 0, pos, errors.New("truncated varint")
}

// ParseRawFields parses wire-level Protostuff/Protobuf fields from raw bytes.
func ParseRawFields(data []byte) ([]RawField, error) {
	if len(data) == 0 {
		return nil, nil
	}
	fields, _, err := parseFields(data, 0, 0, 0)
	return fields, err
}

func parseFields(buf []byte, pos int, groupField uint64, depth int) ([]RawField, int, error) {
	if depth > maxGroupDepth {
		return nil, pos, errors.New("maximum group nesting depth exceeded")
	}

	var fields []RawField
	end := len(buf)

	for pos < end {
		tagVal, nextPos, err := DecodeVarint(buf, pos)
		if err != nil {
			return nil, pos, err
		}
		pos = nextPos
		fieldNum := tagVal >> 3
		wireType := uint8(tagVal & 7)
		if fieldNum == 0 {
			return nil, pos, fmt.Errorf("invalid tag 0 at offset %d", pos)
		}

		switch wireType {
		case WireVarint:
			val, np, err := DecodeVarint(buf, pos)
			if err != nil {
				return nil, pos, err
			}
			pos = np
			fields = append(fields, RawField{Tag: fieldNum, WireType: wireType, Varint: val})

		case Wire64Bit:
			if pos+8 > end {
				return nil, pos, fmt.Errorf("truncated 64-bit field %d", fieldNum)
			}
			raw := make([]byte, 8)
			copy(raw, buf[pos:pos+8])
			pos += 8
			fields = append(fields, RawField{Tag: fieldNum, WireType: wireType, Bytes: raw})

		case WireLen:
			sizeVal, np, err := DecodeVarint(buf, pos)
			if err != nil {
				return nil, pos, err
			}
			pos = np
			if sizeVal > uint64(end-pos) {
				return nil, pos, fmt.Errorf("truncated length-delimited field %d (need %d, have %d)", fieldNum, sizeVal, end-pos)
			}
			size := int(sizeVal)
			raw := make([]byte, size)
			copy(raw, buf[pos:pos+size])
			pos += size
			fields = append(fields, RawField{Tag: fieldNum, WireType: wireType, Bytes: raw})

		case Wire32Bit:
			if pos+4 > end {
				return nil, pos, fmt.Errorf("truncated 32-bit field %d", fieldNum)
			}
			raw := make([]byte, 4)
			copy(raw, buf[pos:pos+4])
			pos += 4
			fields = append(fields, RawField{Tag: fieldNum, WireType: wireType, Bytes: raw})

		case WireStartGroup:
			nested, np, err := parseFields(buf, pos, fieldNum, depth+1)
			if err != nil {
				return nil, pos, err
			}
			pos = np
			fields = append(fields, RawField{Tag: fieldNum, WireType: wireType, Group: nested})

		case WireEndGroup:
			if groupField == 0 {
				return fields, pos, nil
			}
			if fieldNum != groupField {
				return nil, pos, fmt.Errorf("mismatched END_GROUP %d != %d", fieldNum, groupField)
			}
			return fields, pos, nil

		default:
			return nil, pos, fmt.Errorf("unsupported wire type %d for field %d", wireType, fieldNum)
		}
	}

	if groupField != 0 {
		return nil, pos, fmt.Errorf("unterminated group %d", groupField)
	}
	return fields, pos, nil
}

// ExtractPolyClassName returns the Java FQCN stored in Protostuff tag 127, if present.
func ExtractPolyClassName(fields []RawField) string {
	for _, f := range fields {
		if f.Tag == PolyClassTag && f.WireType == WireLen {
			return string(f.Bytes)
		}
	}
	return ""
}

// DecodeMapEntry decodes a Protostuff MapSchema entry (tag 1 = key, tag 2 = value).
func DecodeMapEntry(raw []byte) (string, any, error) {
	sub, err := ParseRawFields(raw)
	if err != nil {
		return "", nil, err
	}
	var key string
	var value any
	for _, f := range sub {
		switch f.Tag {
		case 1:
			if f.WireType == WireLen {
				key = string(f.Bytes)
			}
		case 2:
			switch f.WireType {
			case WireLen:
				if isPrintableUTF8(f.Bytes) {
					value = string(f.Bytes)
				} else {
					dyn, err := DecodeDynamicValue(f.Bytes)
					if err != nil {
						value = hex.EncodeToString(f.Bytes)
					} else {
						value = dyn
					}
				}
			case WireVarint:
				value = f.Int64()
			default:
				value = hex.EncodeToString(f.Bytes)
			}
		}
	}
	return key, value, nil
}

// DecodeDynamicValue decodes a Protostuff polymorphic boxed primitive ObjectSchema value.
// Tags: 1=bool, 2=byte, 3=char, 4=short, 5=int32, 6=int64, 7=float, 8=double, 9=string, 10=bytes.
func DecodeDynamicValue(raw []byte) (any, error) {
	sub, err := ParseRawFields(raw)
	if err != nil {
		return nil, err
	}
	if len(sub) == 0 {
		return nil, nil
	}
	f := sub[0]
	switch {
	case f.Tag == 1 && f.WireType == WireVarint:
		return f.Bool(), nil
	case (f.Tag == 2 || f.Tag == 4 || f.Tag == 5) && f.WireType == WireVarint:
		return f.Int32(), nil
	case f.Tag == 6 && f.WireType == WireVarint:
		return f.Int64(), nil
	case f.Tag == 7 && f.WireType == Wire32Bit:
		return f.Float32()
	case f.Tag == 8 && f.WireType == Wire64Bit:
		return f.Float64()
	case f.Tag == 9 && f.WireType == WireLen:
		return string(f.Bytes), nil
	case f.WireType == WireLen:
		if utf8.Valid(f.Bytes) {
			return string(f.Bytes), nil
		}
		return hex.EncodeToString(f.Bytes), nil
	default:
		return f.Varint, nil
	}
}

func isPrintableUTF8(b []byte) bool {
	if len(b) == 0 || !utf8.Valid(b) {
		return false
	}
	for _, r := range string(b) {
		if r < 32 && r != '\r' && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}

// Encoder builds standard Protobuf/Protostuff wire-format payloads.
type Encoder struct {
	buf []byte
}

// NewEncoder creates an empty Protostuff wire encoder.
func NewEncoder() *Encoder {
	return &Encoder{buf: make([]byte, 0, 64)}
}

// AppendInt32 appends a signed 32-bit integer as a 64-bit two's complement varint.
func (e *Encoder) AppendInt32(tag uint64, v int32) *Encoder {
	return e.AppendInt64(tag, int64(v))
}

// AppendInt64 appends a signed 64-bit integer as a varint.
func (e *Encoder) AppendInt64(tag uint64, v int64) *Encoder {
	e.buf = AppendVarint(e.buf, (tag<<3)|uint64(WireVarint))
	e.buf = AppendVarint(e.buf, uint64(v))
	return e
}

// AppendBool appends a boolean as a varint (0 or 1).
func (e *Encoder) AppendBool(tag uint64, v bool) *Encoder {
	var n uint64
	if v {
		n = 1
	}
	e.buf = AppendVarint(e.buf, (tag<<3)|uint64(WireVarint))
	e.buf = AppendVarint(e.buf, n)
	return e
}

// AppendString appends a length-delimited UTF-8 string.
func (e *Encoder) AppendString(tag uint64, s string) *Encoder {
	e.buf = AppendVarint(e.buf, (tag<<3)|uint64(WireLen))
	e.buf = AppendVarint(e.buf, uint64(len(s)))
	e.buf = append(e.buf, s...)
	return e
}

// AppendBytes appends a length-delimited byte slice.
func (e *Encoder) AppendBytes(tag uint64, b []byte) *Encoder {
	e.buf = AppendVarint(e.buf, (tag<<3)|uint64(WireLen))
	e.buf = AppendVarint(e.buf, uint64(len(b)))
	e.buf = append(e.buf, b...)
	return e
}

// AppendMessage appends an already-encoded nested message as length-delimited bytes.
func (e *Encoder) AppendMessage(tag uint64, encoded []byte) *Encoder {
	return e.AppendBytes(tag, encoded)
}

// Bytes returns a copy of the encoded wire bytes.
func (e *Encoder) Bytes() []byte {
	out := make([]byte, len(e.buf))
	copy(out, e.buf)
	return out
}
