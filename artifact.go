package artifact

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// ArtifactRef is the canonical Phase-1 durable reference (spec §4.2).
// Field order is normative: Namespace, SchemaVersion, ArtifactType, Digest.
// Identity is exactly (Namespace, SchemaVersion, ArtifactType, Digest);
// the digest alone is not sufficient.
type ArtifactRef struct {
	Namespace     string
	SchemaVersion string
	ArtifactType  string
	Digest        [32]byte
}

// isIdentifierChar reports whether c is in [a-z0-9._-].
func isIdentifierChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-'
}

// ValidateNamespace checks the ASCII-lowercase namespace grammar (spec §5.1):
// one to sixty-four characters from [a-z0-9][a-z0-9._-]{0,63}.
func ValidateNamespace(ns string) error {
	if len(ns) < 1 || len(ns) > 64 {
		return fmt.Errorf("artifact: namespace length %d out of range [1,64]", len(ns))
	}
	first := ns[0]
	if !(first >= 'a' && first <= 'z' || first >= '0' && first <= '9') {
		return fmt.Errorf("artifact: invalid namespace first character %q", first)
	}
	for i := 0; i < len(ns); i++ {
		if !isIdentifierChar(ns[i]) {
			return fmt.Errorf("artifact: invalid namespace character %q", ns[i])
		}
	}
	return nil
}

// ValidateArtifactType checks the artifact-type grammar (spec §5.2): one or more
// lowercase path segments separated by '/', each matching the namespace identifier
// grammar, maximum encoded UTF-8 length 128 bytes.
func ValidateArtifactType(t string) error {
	if len(t) == 0 || len(t) > 128 {
		return fmt.Errorf("artifact: artifact-type length %d out of range [1,128]", len(t))
	}
	for _, seg := range strings.Split(t, "/") {
		if len(seg) < 1 || len(seg) > 64 {
			return fmt.Errorf("artifact: invalid artifact-type segment %q", seg)
		}
		first := seg[0]
		if !(first >= 'a' && first <= 'z' || first >= '0' && first <= '9') {
			return fmt.Errorf("artifact: invalid artifact-type segment start %q", seg)
		}
		for i := 0; i < len(seg); i++ {
			if !isIdentifierChar(seg[i]) {
				return fmt.Errorf("artifact: invalid artifact-type character %q in %q", seg[i], seg)
			}
		}
	}
	return nil
}

// SchemaVersion1 is the single Phase-1 schema version (spec §5.3).
const SchemaVersion1 = "1"

// ValidateSchemaVersion checks the schema-version grammar (spec §5.3).
// Phase-1 canonical artifacts use exactly "1".
func ValidateSchemaVersion(v string) error {
	if v != SchemaVersion1 {
		return fmt.Errorf("artifact: unsupported schema version %q (Phase 1 requires %q)", v, SchemaVersion1)
	}
	return nil
}

// ComputeRef computes the canonical ArtifactRef for payload P (spec §4.3):
//
//	Header = Len32(ns) || ns || Len32(ver) || ver || Len32(type) || type
//	Digest = SHA-256(Header || P)
//
// There is exactly one Phase-1 artifact digest formula.
func ComputeRef(namespace, schemaVersion, artifactType string, payload []byte) (ArtifactRef, error) {
	if err := ValidateNamespace(namespace); err != nil {
		return ArtifactRef{}, err
	}
	if err := ValidateSchemaVersion(schemaVersion); err != nil {
		return ArtifactRef{}, err
	}
	if err := ValidateArtifactType(artifactType); err != nil {
		return ArtifactRef{}, err
	}
	ns, err := EncodeString(namespace)
	if err != nil {
		return ArtifactRef{}, err
	}
	ver, err := EncodeString(schemaVersion)
	if err != nil {
		return ArtifactRef{}, err
	}
	typ, err := EncodeString(artifactType)
	if err != nil {
		return ArtifactRef{}, err
	}
	h := sha256.New()
	h.Write(ns)
	h.Write(ver)
	h.Write(typ)
	h.Write(payload)
	var digest [32]byte
	copy(digest[:], h.Sum(nil))
	return ArtifactRef{
		Namespace:     namespace,
		SchemaVersion: schemaVersion,
		ArtifactType:  artifactType,
		Digest:        digest,
	}, nil
}

// Encode returns the canonical embedded ArtifactRef encoding (spec §4.4):
// String(Namespace) || String(SchemaVersion) || String(ArtifactType) || Digest32.
func (r ArtifactRef) Encode() []byte {
	out := MustEncodeString(r.Namespace)
	out = append(out, MustEncodeString(r.SchemaVersion)...)
	out = append(out, MustEncodeString(r.ArtifactType)...)
	return append(out, EncodeDigest32(r.Digest)...)
}

// DecodeArtifactRef decodes one embedded ArtifactRef and validates the
// namespace/type/version grammars.
func DecodeArtifactRef(d *Decoder) (ArtifactRef, error) {
	ns, err := d.String()
	if err != nil {
		return ArtifactRef{}, err
	}
	ver, err := d.String()
	if err != nil {
		return ArtifactRef{}, err
	}
	typ, err := d.String()
	if err != nil {
		return ArtifactRef{}, err
	}
	digest, err := d.Digest32()
	if err != nil {
		return ArtifactRef{}, err
	}
	if err := ValidateNamespace(ns); err != nil {
		return ArtifactRef{}, err
	}
	if err := ValidateSchemaVersion(ver); err != nil {
		return ArtifactRef{}, err
	}
	if err := ValidateArtifactType(typ); err != nil {
		return ArtifactRef{}, err
	}
	return ArtifactRef{Namespace: ns, SchemaVersion: ver, ArtifactType: typ, Digest: digest}, nil
}
