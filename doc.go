// Package artifact is the strictly mechanical foundation of the
// PhysMath/phys-* architecture (plan9.3 §4, spec §§4-5).
//
// It owns: ArtifactRef identity, the single artifact digest formula,
// canonical primitive framing, BigInt encoding, and namespace/type/version
// grammar validation.
//
// It owns NO mathematical vocabulary, NO physical vocabulary, NO protocol
// semantics, and NO generic utilities. See boundary_test.go.
package artifact
