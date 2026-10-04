package artifact

import (
	"fmt"
	"math/big"
)

// BigInt is an exact signed integer for mathematical rational values
// (spec §4.6). Mag holds the minimal unsigned big-endian magnitude;
// empty Mag means zero. Neg must be false when the value is zero.
//
// Canonical encoding:
//
//	Sign:            UInt8   (0x00 non-negative, 0x01 negative)
//	MagnitudeLength: UInt32BE (byte length of the minimal magnitude)
//	Magnitude:       minimal unsigned big-endian bytes, no leading zeros
//
// Zero is 0x00 || UInt32BE(0). Negative zero is invalid.
type BigInt struct {
	Neg bool
	Mag []byte
}

// BigIntFromBig converts a math/big value to canonical BigInt form.
func BigIntFromBig(x *big.Int) BigInt {
	if x == nil || x.Sign() == 0 {
		return BigInt{}
	}
	mag := x.Bytes() // big.Int.Bytes is already minimal unsigned big-endian.
	out := make([]byte, len(mag))
	copy(out, mag)
	return BigInt{Neg: x.Sign() < 0, Mag: out}
}

// BigIntFromInt64 converts an int64 to canonical BigInt form.
func BigIntFromInt64(v int64) BigInt {
	return BigIntFromBig(big.NewInt(v))
}

// ToBig converts back to math/big.
func (b BigInt) ToBig() *big.Int {
	v := new(big.Int).SetBytes(b.Mag)
	if b.Neg {
		v.Neg(v)
	}
	return v
}

// Valid reports whether b is in canonical form: no leading zero bytes in a
// non-empty magnitude, and no negative zero.
func (b BigInt) Valid() bool {
	if len(b.Mag) == 0 {
		return !b.Neg
	}
	return b.Mag[0] != 0x00
}

// Encode returns the canonical byte encoding. Non-canonical values
// (leading zeros, negative zero) are rejected.
func (b BigInt) Encode() ([]byte, error) {
	if !b.Valid() {
		return nil, fmt.Errorf("artifact: non-canonical BigInt")
	}
	out := EncodeUInt8(0x00)
	if b.Neg {
		out = EncodeUInt8(0x01)
	}
	out = append(out, EncodeUInt32(uint32(len(b.Mag)))...)
	return append(out, b.Mag...), nil
}

// DecodeBigInt decodes one canonical BigInt and rejects negative zero and
// redundant leading zero bytes.
func DecodeBigInt(d *Decoder) (BigInt, error) {
	sign, err := d.UInt8()
	if err != nil {
		return BigInt{}, err
	}
	var neg bool
	switch sign {
	case 0x00:
	case 0x01:
		neg = true
	default:
		return BigInt{}, fmt.Errorf("artifact: invalid BigInt sign 0x%02x", sign)
	}
	n, err := d.UInt32()
	if err != nil {
		return BigInt{}, err
	}
	raw, err := d.take(int(n))
	if err != nil {
		return BigInt{}, err
	}
	mag := make([]byte, len(raw))
	copy(mag, raw)
	b := BigInt{Neg: neg, Mag: mag}
	if len(mag) == 0 {
		if neg {
			return BigInt{}, fmt.Errorf("artifact: negative zero BigInt")
		}
		return b, nil
	}
	if mag[0] == 0x00 {
		return BigInt{}, fmt.Errorf("artifact: non-minimal BigInt magnitude")
	}
	return b, nil
}
