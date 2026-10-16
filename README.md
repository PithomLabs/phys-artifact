# PhysArtifact — canonical artifact identity substrate

`github.com/PithomLabs/phys-artifact` is the strictly mechanical
foundation of the Phys architecture. It gives every artifact a stable
identity and canonical bytes. It understands no mathematics, no
physics, and no scientific protocols.

## What it owns

- **Artifact references** (`ArtifactRef`): namespace, type, schema
  version, and digest (`artifact.go`).
- **The single artifact digest formula** used for content addressing.
- **Canonical primitive framing**: strings, integers, sets,
  sequences, and optional values with deterministic field order
  (`primitives.go`).
- **Big integers** (`BigInt`): construction, conversion, and
  canonical encoding (`bigint.go`).
- **Grammar validation** for namespaces, artifact types, and versions.

The contract is stated in `doc.go` and enforced by
`boundary_test.go`: this package owns no mathematical vocabulary, no
physical vocabulary, no protocol semantics, and no generic utilities.

## What it explicitly does not own

Anything that requires understanding what an artifact *means*.
Expression shapes, physical assumptions, theorem status, derivation
validity, and scientific truth all live in higher layers. A stable
digest proves stable identity of a representation — nothing more.

## How it fits into Phys

```text
phys-lib/<framework>   physical meaning (assumptions, equations, theorems)
        ↓ mints into stores, cites by reference
phys-math              pure mathematical representation + bounded execution
        ↓ encodes through
phys-artifact          canonical bytes + identity (this repository)
```

`phys-math` builds canonical mathematical artifacts on top of these
primitives; `phys-lib` framework packages mint their assumptions,
equations, and theorems into content-addressed stores.
[PhysTrace](https://github.com/PithomLabs/phys-trace) coordinates
paper-derivation workflows that consume these artifacts.

Related repositories:
[phys-math](https://github.com/PithomLabs/phys-math),
[phys-lib](https://github.com/PithomLabs/phys-lib).

## How do I start?

```bash
go build ./...
go test ./...
```

Read `doc.go` first (the ownership contract), then `artifact.go`
(identity), `primitives.go` (framing), and `bigint.go`. Read
`boundary_test.go` to see what the package refuses to know.

Contributor guidance: this repository has no separate `AGENTS.md`;
the contract is `doc.go` plus the boundary tests. Cross-repository
workflow rules live in the collective
[phys-system manual](https://github.com/PithomLabs/phys-system/blob/main/AGENTS.md).

## Derivations

Scientific derivation formalization is governed by the normative
[Scientific Derivation Specification Guidelines](https://github.com/PithomLabs/phys-trace/blob/main/DERIVATION_SPEC_GUIDELINES.md),
not by this README. This package's role in that process is narrow:
deterministic identity and encoding, so that higher layers can cite
exact bytes. Canonical bytes are not truth, matching digests are not
proof, and a successful round-trip establishes representation
stability only.

## License 

MIT
