package artifact

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"unicode/utf8"
)

// Canonical primitive framing (spec §4.5).
//
//	String(T)   = UInt32BE(byte_length(T)) || UTF-8(T)
//	Bytes(B)    = UInt32BE(len(B)) || B
//	UInt8       = 1 byte unsigned integer
//	UInt16      = 2-byte unsigned big-endian integer
//	UInt32      = 4-byte unsigned big-endian integer
//	UInt64      = 8-byte unsigned big-endian integer
//	Bool        = 0x00 or 0x01
//	Digest32    = exactly 32 bytes
//	Optional(T) = 0x00 when absent; 0x01 || CanonicalEncoding(T) when present
//	Sequence(T) = UInt32BE(count) || elements in declared order
//	Set(T)      = UInt32BE(count) || elements sorted by canonical element bytes
//	Enum        = String(canonical-token)
//	TaggedUnion = UInt8(discriminator) || selected variant payload

// EncodeString encodes s with length-prefixed framing. s must be valid UTF-8.
func EncodeString(s string) ([]byte, error) {
	if !utf8.ValidString(s) {
		return nil, fmt.Errorf("artifact: string is not valid UTF-8")
	}
	return append(EncodeUInt32(uint32(len(s))), s...), nil
}

// MustEncodeString encodes s and panics on invalid UTF-8. Use only for
// compile-time-known-valid constants in tests and fixtures.
func MustEncodeString(s string) []byte {
	b, err := EncodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// EncodeBytes encodes b with length-prefixed framing.
func EncodeBytes(b []byte) []byte {
	out := EncodeUInt32(uint32(len(b)))
	return append(out, b...)
}

// EncodeUInt8 encodes a single unsigned byte.
func EncodeUInt8(v uint8) []byte { return []byte{v} }

// EncodeUInt16 encodes a 2-byte unsigned big-endian integer.
func EncodeUInt16(v uint16) []byte {
	var out [2]byte
	binary.BigEndian.PutUint16(out[:], v)
	return out[:]
}

// EncodeUInt32 encodes a 4-byte unsigned big-endian integer.
func EncodeUInt32(v uint32) []byte {
	var out [4]byte
	binary.BigEndian.PutUint32(out[:], v)
	return out[:]
}

// EncodeUInt64 encodes an 8-byte unsigned big-endian integer.
func EncodeUInt64(v uint64) []byte {
	var out [8]byte
	binary.BigEndian.PutUint64(out[:], v)
	return out[:]
}

// EncodeBool encodes false as 0x00 and true as 0x01.
func EncodeBool(v bool) []byte {
	if v {
		return []byte{0x01}
	}
	return []byte{0x00}
}

// EncodeDigest32 encodes exactly 32 digest bytes.
func EncodeDigest32(d [32]byte) []byte {
	out := make([]byte, 32)
	copy(out, d[:])
	return out
}

// EncodeOptional encodes presence-framed payload: 0x00 when absent,
// 0x01 || payload when present.
func EncodeOptional(present bool, payload []byte) []byte {
	if !present {
		return []byte{0x00}
	}
	return append([]byte{0x01}, payload...)
}

// EncodeSequence encodes pre-encoded elements in declared order.
func EncodeSequence(elems [][]byte) []byte {
	out := EncodeUInt32(uint32(len(elems)))
	for _, e := range elems {
		out = append(out, e...)
	}
	return out
}

// EncodeSet encodes pre-encoded elements sorted by bytewise lexicographic
// comparison of their exact canonical encodings. Duplicate canonical elements
// are rejected.
func EncodeSet(elems [][]byte) ([]byte, error) {
	cp := make([][]byte, len(elems))
	copy(cp, elems)
	sort.Slice(cp, func(i, j int) bool { return bytes.Compare(cp[i], cp[j]) < 0 })
	for i := 1; i < len(cp); i++ {
		if bytes.Equal(cp[i-1], cp[i]) {
			return nil, fmt.Errorf("artifact: duplicate canonical element in Set")
		}
	}
	return EncodeSequence(cp), nil
}

// EncodeEnum encodes a closed-vocabulary token as a String. Token validity
// against a particular closed vocabulary is the caller's responsibility;
// this function enforces non-empty valid UTF-8.
func EncodeEnum(token string) ([]byte, error) {
	if token == "" {
		return nil, fmt.Errorf("artifact: empty enum token")
	}
	return EncodeString(token)
}

// EncodeTaggedUnion encodes UInt8(discriminator) || payload.
func EncodeTaggedUnion(discriminator uint8, payload []byte) []byte {
	return append([]byte{discriminator}, payload...)
}

// Decoder is a strict cursor reader for canonical bytes. Any truncation,
// overrun, or malformed framing is an error; trailing bytes must be checked
// with Exhausted by the caller.
type Decoder struct {
	buf []byte
	off int
}

// NewDecoder returns a Decoder over b.
func NewDecoder(b []byte) *Decoder { return &Decoder{buf: b} }

// Offset returns the current read offset.
func (d *Decoder) Offset() int { return d.off }

// Exhausted reports whether all input bytes were consumed.
func (d *Decoder) Exhausted() bool { return d.off == len(d.buf) }

func (d *Decoder) take(n int) ([]byte, error) {
	if n < 0 || d.off+n > len(d.buf) {
		return nil, fmt.Errorf("artifact: truncated canonical input")
	}
	out := d.buf[d.off : d.off+n]
	d.off += n
	return out, nil
}

// UInt8 decodes one unsigned byte.
func (d *Decoder) UInt8() (uint8, error) {
	b, err := d.take(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

// UInt16 decodes a 2-byte unsigned big-endian integer.
func (d *Decoder) UInt16() (uint16, error) {
	b, err := d.take(2)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

// UInt32 decodes a 4-byte unsigned big-endian integer.
func (d *Decoder) UInt32() (uint32, error) {
	b, err := d.take(4)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b), nil
}

// UInt64 decodes an 8-byte unsigned big-endian integer.
func (d *Decoder) UInt64() (uint64, error) {
	b, err := d.take(8)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(b), nil
}

// Bool decodes 0x00 as false and 0x01 as true; any other byte is invalid.
func (d *Decoder) Bool() (bool, error) {
	v, err := d.UInt8()
	if err != nil {
		return false, err
	}
	switch v {
	case 0x00:
		return false, nil
	case 0x01:
		return true, nil
	default:
		return false, fmt.Errorf("artifact: invalid Bool byte 0x%02x", v)
	}
}

// String decodes length-prefixed UTF-8 and rejects invalid UTF-8.
func (d *Decoder) String() (string, error) {
	n, err := d.UInt32()
	if err != nil {
		return "", err
	}
	b, err := d.take(int(n))
	if err != nil {
		return "", err
	}
	if !utf8.Valid(b) {
		return "", fmt.Errorf("artifact: string bytes are not valid UTF-8")
	}
	return string(b), nil
}

// Bytes decodes length-prefixed raw bytes.
func (d *Decoder) Bytes() ([]byte, error) {
	n, err := d.UInt32()
	if err != nil {
		return nil, err
	}
	b, err := d.take(int(n))
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out, nil
}

// Digest32 decodes exactly 32 bytes.
func (d *Decoder) Digest32() ([32]byte, error) {
	var out [32]byte
	b, err := d.take(32)
	if err != nil {
		return out, err
	}
	copy(out[:], b)
	return out, nil
}

// OptionalPresence decodes the one-byte presence discriminator.
func (d *Decoder) OptionalPresence() (bool, error) {
	v, err := d.UInt8()
	if err != nil {
		return false, err
	}
	switch v {
	case 0x00:
		return false, nil
	case 0x01:
		return true, nil
	default:
		return false, fmt.Errorf("artifact: invalid Optional discriminator 0x%02x", v)
	}
}

// SequenceCount decodes the four-byte element count for Sequence/Set.
func (d *Decoder) SequenceCount() (int, error) {
	n, err := d.UInt32()
	if err != nil {
		return 0, err
	}
	return int(n), nil
}
