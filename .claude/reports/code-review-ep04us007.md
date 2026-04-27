# Code Review — EP04US-007 Profile Validation Command

**Feature:** ep04us007
**Reviewer:** @code-reviewer
**Date:** 2026-04-27
**Dimensions:** Architecture conformity · Go idioms & naming · Code-smell metrics · Test consistency
**Requirements:** `docs/ep04/epic-04-requirements.md` (US-007 line 459-494, RF-013 line 174-181, RNF-006 line 238-243)
**Plan:** `.claude/plans/ep04us007/plan.md`

---

## Verdict

**PASS WITH WARNINGS** — 0 BLOCKERs, 4 WARNINGs, 5 INFOs.

All five acceptance scenarios pass. `go vet`, `gofmt`, and `go test ./...` are clean (27/27 packages). The implementation matches the plan; the deviations are noted in the WARNING/INFO sections.

---

## Dimension 1 — Architectural Conformity

### Layering

- Domain port `internal/domain/usecase/profilevalidateuc/port.go` declares the `UseCase` interface with single-method ISP (`Execute`). Doc-comment captures the multi-step contract. CONFORMS.
- VOs `validateProfileInput.go` / `validateProfileOutput.go` live under `internal/domain/vo/profile/` — same package as the existing classification/init VOs. No yaml/json tags. CONFORMS.
- Application impl in `internal/application/usecase/profilevalidateuc/usecase.go` exposes `*Impl` and `New(...)`. Compile-time assertion `var _ profilevalidateucport.UseCase = (*Impl)(nil)` at usecase.go:320. CONFORMS.
- Cobra command `profileValidateCmd.go` consumes the domain interface, not `*Impl`. CONFORMS.
- Wire registers `validateProfileUC := appprofilevalidateuc.New(profileBundleLoader, logger)` and exposes `ValidateProfile profilevalidateuc.UseCase` on `Deps`. CONFORMS.
- New error sentinel and typed error appended to `internal/domain/errors/errors.go` with comment-banner `// EP04US-007 sentinels.`. `*ProfileValidationError.Is` correctly chains to `ErrProfileInvalid`. CONFORMS.

### W-001 — Application layer imports `gopkg.in/yaml.v3` (documented exception) [WARNING]

**File:** `internal/application/usecase/profilevalidateuc/usecase.go:21`

`internal/application/` is forbidden from importing `gopkg.in/yaml.v3` per `application-layer-guidelines.yml:54`. The plan documents the exception in §8 Q2: walking raw `yaml.Node` is the simplest way to detect typo'd classification keys without false-positives on forward-compatible fields the loader's `KnownFields(false)` mode tolerates.

This is the **only** application-layer file in the entire repo that imports `yaml.v3` (verified via `grep`). The deviation is contained, documented in the package doc-comment (lines 4-9), and justified.

**Recommendation:** Accept as-is for this story. If a second use case ever needs the same trick, extract the node-walker into an infrastructure adapter (`fsprofile.NodeWalker`) exposing a domain port (`profile.ScanProfileYamlNodesPort`).

**Type:** WARNING (architectural deviation, but documented and accepted in plan).

### Testdata location is gitignored

The fixtures under `testdata/ep04us007/*/` are not tracked because the project-wide `.gitignore` excludes `testdata/`. The integration tests work because `fixtureDir` resolves to the working tree at test time. Acceptable given the project convention; no action.

---

## Dimension 2 — Go Idioms & Naming

### W-002 — Stored `*slog.Logger` is never used [WARNING]

**File:** `internal/application/usecase/profilevalidateuc/usecase.go:51, 59`

`Impl.logger` is assigned in `New` but never referenced in any method. Three call sites that *could* benefit from a debug log (each helper that silently swallows an `os.ReadFile` / `yaml.Unmarshal` error at lines 165-171, 219-226, 261-268) currently discard the error without logging.

**Expected:** Either remove the unused field (and drop the `*slog.Logger` parameter from `New`) OR add a minimal `u.logger.Debug(...)` line at the swallowed-error sites so operators can diagnose mysterious "no warnings" outcomes when profile.yaml is malformed.

**Type:** WARNING. Tests pass either way; this is a maintainability concern.

### W-003 — Helper `isAfterNameDecode` is misnamed [WARNING]

**File:** `internal/application/usecase/profilevalidateuc/usecase.go:298-306`

```go
func isAfterNameDecode(err error) bool {
    if errors.Is(err, domerr.ErrProfileYamlMissing) {
        return false
    }
    return true
}
```

The name claims it returns true "when the LoadBundle failure happened after profile.yaml was decoded". In reality it returns true for **every** non-nil error except `ErrProfileYamlMissing` — including a yaml-decode failure that happened **before** name decode. Behaviour is correct (the body of the gated block re-decodes anyway and tolerates errors), but the function name is misleading.

**Expected:** Rename to something descriptive of the actual behaviour (e.g., `loadFailedAfterFileOpen`, `profileYamlReadable`) or invert to `isProfileYamlMissing(err error) bool` and adjust the call site at line 121.

**Type:** WARNING (naming clarity).

### I-001 — Helpers re-read `profile.yaml` three times per Execute [INFO]

**File:** `internal/application/usecase/profilevalidateuc/usecase.go:163-169, 218-225, 260-268`

`scanClassificationKeyTypos`, `scanDuplicateTypeIDs`, and `readRawNameField` each independently call `os.ReadFile(filepath.Join(dir, "profile.yaml"))`. For a real-world profile (~1-3 KB) this is microseconds and cached at OS level, so the practical impact is nil. Architectural note: a shared "profile.yaml bytes" closure read once at the top of `Execute` would simplify the helpers and make a unit test that injects content trivial.

**Recommendation:** Optional. If touched in a future cycle, factor the read into a single `data, err := os.ReadFile(...)` and pass `data` (not `dir`) to the helpers.

**Type:** INFO.

### I-002 — Doc comment on package mentions `bundleDto.go:89-95` line numbers [INFO]

**File:** `internal/application/usecase/profilevalidateuc/usecase.go:30`

> // knownClassificationKeys is the canonical set of field keys allowed inside a
> // types[].classification[] mapping entry. Sourced from bundleDto.go:89-95.

Hard-coded line numbers go stale fast. The reference is correct today but will rot the next time `bundleDto.go` is edited.

**Recommendation:** Replace with `Sourced from bundleClassificationDTO in internal/infrastructure/fsprofile/bundleDto.go.` (no line numbers).

**Type:** INFO.

### I-003 — Filename `profileValidateIntegration_test.go` is camelCase but mixed-case [INFO]

**File:** `internal/cli/command/profileValidateIntegration_test.go`

The repo convention requires camelCase filenames (`CLAUDE.md` rule). The chosen name is grammatically `profileValidateIntegrationTest.go` collapsed, but the trailing `_test.go` is the toolchain-required suffix. Conformant. Recorded as INFO so future reviewers know the suffix is intentional.

**Type:** INFO.

---

## Dimension 3 — Code-Smell Metrics

| Metric | File | Value | Threshold | Verdict |
|---|---|---|---|---|
| LOC | usecase.go | 320 | <500 | PASS |
| LOC | usecase_test.go | 320 | <500 | PASS |
| LOC | profileValidateCmd.go | 37 | <100 | PASS |
| LOC | profileValidateIntegration_test.go | 147 | <500 | PASS |
| Cyclomatic | `Execute` | ~9 | <15 | PASS |
| Cyclomatic | `scanClassificationKeyTypos` | ~7 | <15 | PASS |
| Cyclomatic | `scanDuplicateTypeIDs` | ~6 | <15 | PASS |
| Function count in `format.TranslateError` | errors.go | 1 (long) | informational | INFO (existing) |

`go vet ./...` clean. `gofmt -l .` clean. No `panic`, no `init`, no globally mutable state.

### I-004 — `scanClassificationKeyTypos` and `scanDuplicateTypeIDs` share boilerplate [INFO]

**File:** `internal/application/usecase/profilevalidateuc/usecase.go:163-214, 218-256`

Both helpers replicate the same opening:

```go
data, err := os.ReadFile(filepath.Join(dir, "profile.yaml"))
if err != nil { return nil }
var root yaml.Node
if err := yaml.Unmarshal(data, &root); err != nil { return nil }
if root.Kind != yaml.DocumentNode || len(root.Content) == 0 { return nil }
topMap := root.Content[0]
if topMap.Kind != yaml.MappingNode { return nil }
typesSeq := mappingValue(topMap, "types")
if typesSeq == nil || typesSeq.Kind != yaml.SequenceNode { return nil }
```

A small helper `decodeTypesSequence(dir string) (*yaml.Node, bool)` would eliminate ~16 duplicated lines.

**Type:** INFO. Defer to a future refactor.

---

## Dimension 4 — Test Consistency

### Coverage of acceptance scenarios

Mapping requirements ↔ tests:

| Scenario (epic-04 line) | Use case unit test | Integration test |
|---|---|---|
| 1. Clean profile, exit 0 | `TestValidateUC_CleanProfile` | `TestProfileValidate_CleanProfile_ExitsZero` |
| 2. Missing `name`, exit 1 | `TestValidateUC_MissingName` | `TestProfileValidate_MissingName_ExitsOne` |
| 3. Missing template, exit 1 | `TestValidateUC_MissingTemplate` | `TestProfileValidate_MissingTemplate_ExitsOne` |
| 4. Unknown classification, warn (exit 0) | `TestValidateUC_UnknownClassificationField_IsWarning` | `TestProfileValidate_UnknownClassificationField_WarnsButPasses` |
| 5. Duplicate type ids, exit 1 | `TestValidateUC_DuplicateTypeIDs` | `TestProfileValidate_DuplicateTypeIds_ExitsOne` |

Plus two extra unit tests:
- `TestValidateUC_PathDoesNotExist` (RF-013 line 174-181 "path that doesn't exist causes immediate exit 1").
- `TestValidateUC_FatalsAreSorted` (determinism for stderr / asserts).
- `TestValidateUC_WarningsPresentWhenFatalPresent` (combined warning + fatal scenario).

CONFORMS to RNF-006 (deterministic ordering).

### W-004 — Integration test asserts on `err.Error()` instead of typed unwrap [WARNING]

**File:** `internal/cli/command/profileValidateIntegration_test.go:76-81`

```go
err := cmd.ExecuteContext(context.Background())
require.Error(t, err)
// format.TranslateError wraps ProfileValidationError into errors.New(string),
// so errors.As cannot find the typed error at this point. Assert on the
// rendered message string which carries the canonical literal.
require.Contains(t, err.Error(), "missing required field: name")
```

The comment correctly explains that `format.TranslateError` returns `errors.New(b.String())` at format/errors.go:98 — this **drops** the typed error, breaking `errors.As`/`errors.Is` for downstream test assertions. While the test works, the design choice means consumers of `format.TranslateError` cannot use idiomatic Go error inspection on this code path.

**Expected:** Either:
1. Have `format.TranslateError` return `fmt.Errorf("%s: %w", b.String(), pve)` so the typed error is preserved, OR
2. Add a helper test method on `ProfileValidationError` and call it directly at the use case layer (already exercised by the unit tests).

**Risk:** Low — the integration tests do work today. Future tests that want to assert on `pve.Path` or `pve.Errors` from a cobra-driven flow will be unable to.

**Type:** WARNING. Out of scope for this story; flag for follow-up.

### Test isolation

- All unit tests use `t.Parallel()` and `t.TempDir()`. No shared global state.
- All integration tests use `bytes.Buffer` for stdout/stderr (per project convention).
- The `nopWriter` for slog suppression is local to `profileValidateIntegration_test.go` rather than reused — minor duplication with similar helpers in other `*Integration_test.go`s but consistent style.

### I-005 — `containsStr` is a thin wrapper around `strings.Contains` [INFO]

**File:** `internal/application/usecase/profilevalidateuc/usecase_test.go:317-319`

```go
func containsStr(s, substr string) bool {
    return strings.Contains(s, substr)
}
```

Adds nothing over the stdlib call. Three call sites can inline `strings.Contains(...)` directly.

**Type:** INFO.

---

## Summary

| Severity | Count | IDs |
|---|---|---|
| BLOCKER | 0 | — |
| WARNING | 4 | W-001, W-002, W-003, W-004 |
| INFO | 5 | I-001, I-002, I-003, I-004, I-005 |

No BLOCKERs. WARNINGs and INFOs are recorded for awareness; per the QA contract they are surfaced in the verdict but not auto-fixed.

**Overall:** PASS WITH WARNINGS. Feature meets all acceptance criteria, `go test ./...` is green, `go vet` and `gofmt` are clean. Deviations are contained, documented, and acceptable.
