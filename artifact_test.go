package artifact

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"testing"
)

// §25.1: identical payloads + identity → identical digests (twice computed).
func TestIdentityDeterminism(t *testing.T) {
	payload := append(MustEncodeString("display"), EncodeUInt32(42)...)
	a, err := ComputeRef("physmath", "1", "expression/1", payload)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ComputeRef("physmath", "1", "expression/1", payload)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("digests differ: %x vs %x", a.Digest, b.Digest)
	}
}

// Independent reimplementation of the §4.3 digest framing (hand-rolled, no helpers).
func TestDigestFormulaIndependent(t *testing.T) {
	ns, ver, typ := "phys-gr", "1", "assumption/1"
	payload := []byte{0x01, 0x02, 0x03}
	ref, err := ComputeRef(ns, ver, typ, payload)
	if err != nil {
		t.Fatal(err)
	}
	var header []byte
	for _, s := range []string{ns, ver, typ} {
		var l [4]byte
		binary.BigEndian.PutUint32(l[:], uint32(len(s)))
		header = append(header, l[:]...)
		header = append(header, s...)
	}
	want := sha256.Sum256(append(header, payload...))
	if ref.Digest != want {
		t.Fatalf("digest mismatch: got %x want %x", ref.Digest, want)
	}
}

// §25.2: framing keeps ambiguous concatenations distinct.
func TestPrimitiveFramingAmbiguity(t *testing.T) {
	a1, _ := EncodeString("ab")
	a2, _ := EncodeString("c")
	b1, _ := EncodeString("a")
	b2, _ := EncodeString("bc")
	x := append(append([]byte{}, a1...), a2...)
	y := append(append([]byte{}, b1...), b2...)
	if bytes.Equal(x, y) {
		t.Fatal("ambiguous payload sequences collide under framing")
	}
	// Optional presence changes bytes even with empty payload.
	if bytes.Equal(EncodeOptional(false, nil), EncodeOptional(true, nil)) {
		t.Fatal("optional absent/present collide")
	}
	// Bool rejects non 0x00/0x01.
	d := NewDecoder([]byte{0x02})
	if _, err := d.Bool(); err == nil {
		t.Fatal("Bool accepted 0x02")
	}
}

func TestSetOrderingAndDuplicates(t *testing.T) {
	one := MustEncodeString("b")
	two := MustEncodeString("a")
	s1, err := EncodeSet([][]byte{one, two})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := EncodeSet([][]byte{two, one})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(s1, s2) {
		t.Fatal("Set order depends on traversal order")
	}
	if _, err := EncodeSet([][]byte{one, one}); err == nil {
		t.Fatal("Set accepted duplicate canonical elements")
	}
}

func TestEmbeddedRefRoundTrip(t *testing.T) {
	ref, err := ComputeRef("physmath", "1", "relation/1", []byte{0x09})
	if err != nil {
		t.Fatal(err)
	}
	enc := ref.Encode()
	dec, err := DecodeArtifactRef(NewDecoder(enc))
	if err != nil {
		t.Fatal(err)
	}
	if dec != ref {
		t.Fatalf("round trip mismatch: %+v vs %+v", dec, ref)
	}
	d := NewDecoder(enc)
	if _, err := DecodeArtifactRef(d); err != nil {
		t.Fatal(err)
	}
	if !d.Exhausted() {
		t.Fatal("trailing bytes after embedded ref")
	}
}

// §25.3: BigInt minimal valid forms accepted; negative zero and leading zeros rejected.
func TestBigIntVectors(t *testing.T) {
	cases := []struct {
		name string
		big  *big.Int
	}{
		{"positive", big.NewInt(123456789)},
		{"negative", big.NewInt(-987654321)},
		{"zero", big.NewInt(0)},
		{"big", new(big.Int).Lsh(big.NewInt(1), 200)},
		{"bigneg", new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 200))},
	}
	for _, c := range cases {
		b := BigIntFromBig(c.big)
		enc, err := b.Encode()
		if err != nil {
			t.Fatalf("%s: encode: %v", c.name, err)
		}
		got, err := DecodeBigInt(NewDecoder(enc))
		if err != nil {
			t.Fatalf("%s: decode: %v", c.name, err)
		}
		if got.ToBig().Cmp(c.big) != 0 {
			t.Fatalf("%s: round trip changed value", c.name)
		}
	}
	// Zero is exactly 0x00 || UInt32BE(0).
	zero, err := BigInt{}.Encode()
	if err != nil {
		t.Fatal(err)
	}
	wantZero := []byte{0x00, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(zero, wantZero) {
		t.Fatalf("zero encoding = %x, want %x", zero, wantZero)
	}
	// Negative zero rejected.
	if _, err := DecodeBigInt(NewDecoder([]byte{0x01, 0x00, 0x00, 0x00, 0x00})); err == nil {
		t.Fatal("negative zero accepted")
	}
	// Redundant leading zero rejected: -1 encoded with magnitude 0x00 0x01.
	if _, err := DecodeBigInt(NewDecoder([]byte{0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01})); err == nil {
		t.Fatal("non-minimal magnitude accepted")
	}
	// Bad sign rejected.
	if _, err := DecodeBigInt(NewDecoder([]byte{0x02, 0x00, 0x00, 0x00, 0x00})); err == nil {
		t.Fatal("bad sign accepted")
	}
}

// Namespace/type/version grammar (§§5.1-5.3, §25.31).
func TestGrammarVectors(t *testing.T) {
	for _, ns := range []string{"physmath", "phys", "phys-gr", "phys-qm", "phys-fixture2", "a", "x9._-"} {
		if err := ValidateNamespace(ns); err != nil {
			t.Fatalf("valid namespace %q rejected: %v", ns, err)
		}
	}
	for _, ns := range []string{"", "Phys", "phys gr", "a-very-long-namespace-that-exceeds-the-sixty-four-character-limit-x", "-lead", ".lead"} {
		if err := ValidateNamespace(ns); err == nil {
			t.Fatalf("invalid namespace %q accepted", ns)
		}
	}
	for _, typ := range []string{"expression/1", "framework/1", "a", "a/b/c"} {
		if err := ValidateArtifactType(typ); err != nil {
			t.Fatalf("valid type %q rejected: %v", typ, err)
		}
	}
	for _, typ := range []string{"", "A/b", "a//b", "/a", "a/"} {
		if err := ValidateArtifactType(typ); err == nil {
			t.Fatalf("invalid type %q accepted", typ)
		}
	}
	if err := ValidateSchemaVersion("1"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSchemaVersion("2"); err == nil {
		t.Fatal("schema version 2 accepted in Phase 1")
	}
	if _, err := ComputeRef("Bad Name", "1", "expression/1", nil); err == nil {
		t.Fatal("bad namespace accepted by ComputeRef")
	}
}

func TestTaggedUnionAndSequenceRoundTrip(t *testing.T) {
	u := EncodeTaggedUnion(0x0A, MustEncodeString("branch"))
	d := NewDecoder(u)
	tag, err := d.UInt8()
	if err != nil || tag != 0x0A {
		t.Fatalf("tag = %x, %v", tag, err)
	}
	s, err := d.String()
	if err != nil || s != "branch" {
		t.Fatalf("payload = %q, %v", s, err)
	}
	if !d.Exhausted() {
		t.Fatal("trailing bytes")
	}
}
