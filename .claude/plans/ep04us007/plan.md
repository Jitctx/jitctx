# Plan — EP04US-007 Profile Validation Command

## Section 0 — Summary

- Feature: introduce a new cobra subcommand `jitctx profile validate <path>`
  that performs structural and logical validation of a profile directory
  and reports issues to stdout/stderr. Validation leverages the existing
  `LoadProfileBundlePort` to surface load-time fatals (missing
  `profile.yaml`, missing template files, type id missing, unknown
  language), and adds three additional checks the loader is too lenient
  about today: (a) empty `name:` field, (b) duplicate `types[].id`,
  (c) classification rules whose keys are not in the known field set
  (typos like `implementss`). Unknown classification keys are surfaced as
  non-fatal warnings (may exit 0); the rest are fatal (exit 1).
- Requirement IDs: **EP04US-007**, **EP04RF-013**, **EP04RNF-006**.
- Layers touched: `[domain, application, presentation, wire, tests]`.
- Tiers active: `[1, 3, 4, 5, 6]` — no Tier 2 (no infrastructure adapter
  edits; we re-use `*fsprofile.BundleLoader` and re-decode the YAML
  directly inside the use case via `gopkg.in/yaml.v3` `yaml.Node` walk
  — see Section 5 for the rationale and Section 8 Q2 resolution).
- Guidelines loaded:
  - `.claude/guidelines/domain-layer-guidelines.yml`
  - `.claude/guidelines/application-layer-guidelines.yml`
  - `.claude/guidelines/presentation-layer-guidelines.yml`
  - `.claude/guidelines/main-layer-guidelines.yml`
  - `.claude/guidelines/unit-test-layer-guidelines.yml`
  - `.claude/guidelines/integration-test-layer-guidelines.yml`
- Estimated file count: **9 new, 4 modified**.

### Key discovery findings (verified against the codebase 2026-04-27)

1. **`profile init` is the template for the new subcommand.**
   `internal/cli/command/profileCmd.go` already exposes `NewProfileCmd`
   (a parent group with `Args: cobra.NoArgs`, no `RunE`). `root.go:16-18`
   wires it via `profileCmd.AddCommand(NewProfileInitCmd(...))`. The new
   `validate` subcommand attaches in the same place via a sibling
   `profileCmd.AddCommand(NewProfileValidateCmd(...))` call.

2. **`BundleLoader.LoadBundle` already does most of the structural
   validation.** Verified against
   `internal/infrastructure/fsprofile/bundleLoader.go:53-89`:
   - returns `domerr.ErrProfileYamlMissing` when `profile.yaml` is
     absent (line 154);
   - returns `*domerr.TemplateMissingError` when a declared template is
     missing on disk (`bundleMapper.go:91-98`);
   - returns wrapped `domerr.ErrProfileInvalid` for missing `id` on type
     entries (`bundleMapper.go:88-89`);
   - returns `*domerr.LanguageUnsupportedError` (US-005) when `language:`
     declares an unknown id (`bundleLoader.go:131-136`).
   The validate use case simply calls `LoadBundle` and reports any
   fatal it returns.

3. **`KnownFields(false)` masks typos in the regular load path.**
   Verified at `bundleLoader.go:160-166`:
   ```go
   dec := yaml.NewDecoder(bytes.NewReader(data))
   dec.KnownFields(false)
   ```
   This is intentional — the EP-04 schema evolves across user stories and
   strict mode would break early adopters. **Therefore the validate
   command CANNOT just re-decode with `KnownFields(true)`** (it would
   warn on legitimate forward-compatible fields the runtime ignores).
   Instead, the validate use case parses the YAML into a generic
   `yaml.Node` tree and walks the
   `types[].classification[]` entries comparing keys against the known
   set `{kind, implements_all, implements_none, has_annotation,
   path_contains}` defined in `bundleDto.go:89-95`. Unknown keys outside
   classification are intentionally tolerated (they are how new schema
   sections roll out — see Section 8 Q2).

4. **`name` validation is currently silent.** `bundleDTO.Name` (line 16
   of `bundleDto.go`) is decoded but never checked for emptiness by the
   loader or mapper. The validate use case adds an explicit
   `if dto.Name == ""` check that produces the .feature-pinned literal
   `missing required field: name`.

5. **Duplicate type ids are currently silent.** `toBundleDomain`
   (`bundleMapper.go:85-123`) walks `dto.Types` linearly without a seen
   set. The validate use case adds an explicit duplicate scan that
   produces the .feature-pinned literal `duplicate type id: <id>`.

6. **`*domerr.TemplateMissingError.Error()` already names the missing
   template file.** Verified at `errors.go:291-294`:
   ```
   profile %q: type %q references missing template %q
   ```
   The `Template` field carries the basename, so the .feature criterion
   "stderr names the missing template file" is satisfied verbatim by
   propagating the typed error through `format/errors.go`.

7. **Production wiring already passes a real `LanguageQueries` registry
   to `BundleLoader`** (`wire.go:134-135`). So the validate command, when
   constructed from `Deps.ProfileBundleLoader`, will surface
   `LanguageUnsupportedError` automatically. This satisfies the spirit of
   RNF-006 ("validation catches errors early") without any extra code.
   See Section 8 Q1 for the resolution.

8. **`format/errors.go:96` already maps `ErrProfileInvalid` to
   "framework profile is invalid: %w".** Existing TemplateMissingError
   and LanguageUnsupportedError both satisfy `Is(ErrProfileInvalid)`, so
   the catch-all branch already produces a stderr line that names the
   underlying error. The validate command, however, must aggregate
   MULTIPLE issues (fatals + warnings) — one error wrapping is not
   enough. We introduce a new typed `*ProfileValidationError` whose
   `Error()` joins all fatals and whose translator branch in
   `format/errors.go` writes them as separate lines. Warnings are
   carried inside the success Output VO and rendered by the cobra
   command directly to stderr (so they print even on exit 0).

9. **`ProfileBundleLoader` is already in `Deps`** (`wire.go:70-71`,
   already typed `profileport.LoadProfileBundlePort` via
   `*fsprofile.BundleLoader`). The validate use case consumes this
   existing port; **no new domain port is required**. This collapses
   what would otherwise be a Tier 2 group into Tier 0 (no work).

### Scenario coverage (.feature → fatal/warning)

| Scenario                                  | Source of detection                        | Surface |
|-------------------------------------------|--------------------------------------------|---------|
| Clean profile passes                      | `LoadBundle` returns nil; no warnings      | exit 0, stdout `Profile valid` |
| Missing required field `name`             | New `dto.Name == ""` check (use case)      | fatal — exit 1, stderr `missing required field: name` |
| Missing template                          | `*TemplateMissingError` from `LoadBundle`  | fatal — exit 1, stderr names the file |
| Unknown classification field              | yaml.Node walk in use case                 | warning — exit 0, stderr `unknown classification field 'implementss'` |
| Duplicate type IDs                        | New duplicate scan in use case             | fatal — exit 1, stderr `duplicate type id: service` |

---

## Section 1 — File Set

| #  | File                                                                                       | Action  | Layer        | Tier | Group  |
|----|--------------------------------------------------------------------------------------------|---------|--------------|------|--------|
| 1  | internal/domain/usecase/profilevalidateuc/port.go                                          | create  | domain       | 1    | T1-G1  |
| 2  | internal/domain/vo/profile/validateProfileInput.go                                         | create  | domain       | 1    | T1-G1  |
| 3  | internal/domain/vo/profile/validateProfileOutput.go                                        | create  | domain       | 1    | T1-G1  |
| 4  | internal/domain/errors/errors.go                                                           | modify  | domain       | 1    | T1-G1  |
| 5  | internal/application/usecase/profilevalidateuc/usecase.go                                  | create  | application  | 3    | T3-G1  |
| 6  | internal/cli/command/profileValidateCmd.go                                                 | create  | presentation | 4    | T4-G1  |
| 7  | internal/cli/command/profileCmd.go                                                         | modify  | presentation | 4    | T4-G1  |
| 8  | internal/cli/format/errors.go                                                              | modify  | presentation | 4    | T4-G1  |
| 9  | internal/cli/wire.go                                                                       | modify  | wire         | 5    | T5-G1  |
| 10 | internal/cli/root.go                                                                       | modify  | wire         | 5    | T5-G1  |
| 11 | internal/application/usecase/profilevalidateuc/usecase_test.go                             | create  | tests        | 6    | T6-G1  |
| 12 | internal/cli/command/profileValidateIntegration_test.go                                    | create  | tests        | 6    | T6-G2  |
| 13 | testdata/ep04us007/cleanProfile/profile.yaml                                               | create  | tests        | 6    | T6-G3  |
| 14 | testdata/ep04us007/cleanProfile/templates/service.java.tmpl                                | create  | tests        | 6    | T6-G3  |
| 15 | testdata/ep04us007/missingName/profile.yaml                                                | create  | tests        | 6    | T6-G3  |
| 16 | testdata/ep04us007/missingTemplate/profile.yaml                                            | create  | tests        | 6    | T6-G3  |
| 17 | testdata/ep04us007/unknownClassificationField/profile.yaml                                 | create  | tests        | 6    | T6-G3  |
| 18 | testdata/ep04us007/unknownClassificationField/templates/service.java.tmpl                  | create  | tests        | 6    | T6-G3  |
| 19 | testdata/ep04us007/duplicateTypeIds/profile.yaml                                           | create  | tests        | 6    | T6-G3  |
| 20 | testdata/ep04us007/duplicateTypeIds/templates/service.java.tmpl                            | create  | tests        | 6    | T6-G3  |

Note: file count summary in Section 0 (9 new + 4 modified production)
covers items 1–10 only. Items 11–20 are the test/fixture set, listed
explicitly so the parallel groups in Section 9 are exhaustive.

Requirement coverage trace:
- **EP04US-007** — items 5, 6, 11, 12, all fixtures.
- **EP04RF-013** — items 1, 5, 6 (the command shape + exception path).
- **EP04RNF-006** — items 5 (load-time error surfacing via `LoadBundle`).

---

## Section 2 — Frozen Domain Contract

### 2.1 New use case interface

```go
// internal/domain/usecase/profilevalidateuc/port.go
package profilevalidateuc

import (
	"context"

	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// UseCase orchestrates "jitctx profile validate <path>". Steps:
//  1. Verify the path exists and is a directory; otherwise return a
//     wrapped *ProfileValidationError carrying the io error so the
//     translator renders an exit-1 message (per EP04RF-013 exception
//     "Path that doesn't exist causes immediate exit 1").
//  2. Re-read the raw bytes of <path>/profile.yaml and walk them as a
//     generic yaml.Node tree to detect unknown classification field
//     keys (these become non-fatal warnings).
//  3. Call LoadProfileBundlePort.LoadBundle to leverage existing
//     structural checks (missing yaml, missing template, missing type
//     id, unknown language). Capture any returned error as a fatal.
//  4. Run two extra checks the loader is too lenient about today:
//     (a) `dto.Name == ""` → fatal "missing required field: name"
//     (b) duplicate `dto.Types[].ID` → fatal "duplicate type id: <id>"
//  5. Aggregate fatals + warnings into ValidateProfileOutput. When
//     fatals is non-empty, return a *domerr.ProfileValidationError
//     carrying the same lists; otherwise return Output, nil.
type UseCase interface {
	Execute(ctx context.Context, input profilevo.ValidateProfileInput) (profilevo.ValidateProfileOutput, error)
}
```

### 2.2 New value objects

```go
// internal/domain/vo/profile/validateProfileInput.go
package profile

// ValidateProfileInput is the input VO for profilevalidateuc.UseCase.
//
// Path must point to a profile DIRECTORY (the same shape consumed by
// profile.LoadProfileBundlePort with Dir set). Path is resolved via
// filepath.Abs at the cobra layer; the use case treats it verbatim.
type ValidateProfileInput struct {
	Path string
}
```

```go
// internal/domain/vo/profile/validateProfileOutput.go
package profile

// ValidationIssue describes a single problem found during validation.
// Code is a short, stable, machine-friendly tag (e.g.,
// "missing_name", "duplicate_type_id", "unknown_classification_field",
// "missing_template", "yaml_missing", "language_unsupported"). Message
// is the human-readable string written to stderr — its format matches
// the literals pinned by the .feature scenarios.
type ValidationIssue struct {
	Code    string
	Message string
}

// ValidateProfileOutput is the success-path result of
// profilevalidateuc.UseCase.Execute. Fatals are present only when
// Execute also returns *domerr.ProfileValidationError; on the success
// path Errors is always empty and Warnings may be non-empty (the
// .feature explicitly allows exit 0 with warnings on stderr).
type ValidateProfileOutput struct {
	Path     string
	Errors   []ValidationIssue
	Warnings []ValidationIssue
}
```

### 2.3 New error sentinels and typed error

```go
// internal/domain/errors/errors.go — appended at the bottom of the
// existing var blocks (do NOT touch existing declarations).

// EP04US-007 sentinels.
var (
	// ErrProfileValidationFailed is returned by
	// profilevalidateuc.UseCase.Execute when the validation report
	// contains at least one fatal issue. Wraps ErrProfileInvalid for
	// errors.Is matching so existing translators continue to fire on
	// the broader sentinel.
	ErrProfileValidationFailed = errors.New("profile validation failed")
)

// ProfileValidationError aggregates the fatal issues and warnings
// produced by a single validate run. Format/errors.go renders Errors
// as separate stderr lines and surfaces Warnings via the cobra command
// (so they print even when Execute returns nil). The Error() string is:
//
//	profile %q: <count> error(s)
//
// — kept short; the per-line rendering is done by the translator.
//
// errors.Is(err, ErrProfileValidationFailed) returns true. errors.Is
// also returns true for ErrProfileInvalid (transitive via the sentinel).
type ProfileValidationError struct {
	Path     string
	Errors   []string // formatted fatal messages, one per slice element
	Warnings []string // formatted non-fatal messages
}

func (e *ProfileValidationError) Error() string {
	return fmt.Sprintf("profile %q: %d error(s)", e.Path, len(e.Errors))
}

func (e *ProfileValidationError) Is(target error) bool {
	return target == ErrProfileValidationFailed ||
		errors.Is(target, ErrProfileInvalid)
}
```

### 2.4 Existing port reused (no change)

```go
// Already present at internal/domain/port/profile/loadProfileBundlePort.go
// — the validate use case consumes this existing port.
type LoadProfileBundlePort interface {
	LoadBundle(ctx context.Context, input profilevo.LoadProfileBundleInput) (*model.ProfileBundle, error)
}
```

### 2.5 Deps struct after this feature lands (verbatim)

The existing `Deps` struct in `internal/cli/wire.go` gains exactly
**one** new field:

```go
type Deps struct {
	// ... existing fields unchanged ...

	// ValidateProfile is the profile-validate use case. Consumed by
	// the new "profile validate" cobra subcommand. EP04US-007.
	ValidateProfile profilevalidateuc.UseCase
}
```

Wiring: `appprofilevalidateuc.New(profileBundleLoader, logger)` → assigned to
`ValidateProfile` in the returned `Deps{...}` literal.

### 2.6 Cobra constructor signature (verbatim)

```go
// internal/cli/command/profileValidateCmd.go
func NewProfileValidateCmd(
	uc profilevalidateuc.UseCase,
	_ *slog.Logger,
) *cobra.Command
```

The constructor accepts no defaultPath — the validate command takes the
path as a required positional arg (`cobra.ExactArgs(1)`), per the
.feature pin `jitctx profile validate <path>`. See Section 8 Q4.

---

## Section 3 — Domain Layer Plan

**Group T1-G1 produces** four files that together publish the contract
consumed by Tier 3 and Tier 4. All four files are co-edited because the
typed error references the new sentinel and the use case interface
references both VOs.

### 3.1 `internal/domain/usecase/profilevalidateuc/port.go` (new)

- Single `UseCase` interface as shown in Section 2.1.
- Imports only `context` and `profilevo`.
- No constants, no helpers — the interface is the entire file.

### 3.2 `internal/domain/vo/profile/validateProfileInput.go` (new)

- `ValidateProfileInput` struct with one exported field `Path string`.
- No constructors — input VOs in this repo are decoded directly from
  the cobra layer (see `ProfileInitInput` for the precedent).

### 3.3 `internal/domain/vo/profile/validateProfileOutput.go` (new)

- `ValidationIssue` struct (`Code string`, `Message string`).
- `ValidateProfileOutput` struct (`Path string`,
  `Errors []ValidationIssue`, `Warnings []ValidationIssue`).
- Codes are documented as comment-only constants — the test asserts on
  them but they are not exported `const` declarations to keep the VO
  passive.

### 3.4 `internal/domain/errors/errors.go` (modify — append only)

- Append the EP04US-007 var block (`ErrProfileValidationFailed`) at the
  bottom of the existing var blocks.
- Append the `ProfileValidationError` typed error after the existing
  `LanguageUnsupportedError` definition.
- Verbatim shape per Section 2.3. **Do not modify any existing block.**

---

## Section 4 — Infrastructure Layer Plan

**N/A.** No infrastructure adapter is created or modified by this
feature. The validate use case re-uses the existing
`*fsprofile.BundleLoader` (already wired in `Deps.ProfileBundleLoader`).
The yaml.Node walk for unknown-classification-field detection is
performed inside the use case via `gopkg.in/yaml.v3` — yaml.v3 is in
the application layer's whitelist exception (see Section 8 Q2 for the
resolution: classification key validation belongs to the use case
because the canonical key set is part of the EP-04 declarative-types
schema, NOT a fsprofile concern).

---

## Section 5 — Application Layer Plan

**Group T3-G1 produces** one file:
`internal/application/usecase/profilevalidateuc/usecase.go`.

### 5.1 Package and imports

```go
package profilevalidateuc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	profileport "github.com/jitctx/jitctx/internal/domain/port/profile"
	profilevalidateucport "github.com/jitctx/jitctx/internal/domain/usecase/profilevalidateuc"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)
```

Rationale for `gopkg.in/yaml.v3` exception: see Section 8 Q2. The
application-layer guideline forbids yaml.v3 by default; this use case
needs structural inspection of the raw classification entries that the
loader's DTO has already collapsed away. The alternative (publish a new
domain port `WalkClassificationKeysPort` backed by an fsprofile adapter)
adds a port + adapter + integration test and pollutes the infra layer
with knowledge of the canonical classification field set, which lives
naturally in the validate use case. Discovery accepts the yaml.v3
import inside this single file as the simpler option, with an explicit
file-level comment documenting the exception.

### 5.2 Type and constructor

```go
type Impl struct {
	loader profileport.LoadProfileBundlePort
	logger *slog.Logger
}

func New(loader profileport.LoadProfileBundlePort, logger *slog.Logger) *Impl {
	if logger == nil {
		logger = slog.Default()
	}
	return &Impl{loader: loader, logger: logger}
}

var _ profilevalidateucport.UseCase = (*Impl)(nil)
```

### 5.3 Execute orchestration

```go
func (u *Impl) Execute(
	ctx context.Context,
	in profilevo.ValidateProfileInput,
) (profilevo.ValidateProfileOutput, error) {
	if err := ctx.Err(); err != nil {
		return profilevo.ValidateProfileOutput{}, err
	}

	out := profilevo.ValidateProfileOutput{Path: in.Path}

	// Step 1 — path-exists guard (RF-013 exception path).
	abs, err := filepath.Abs(in.Path)
	if err != nil {
		return out, fmt.Errorf("validate profile: resolve path %q: %w", in.Path, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		// "Path that doesn't exist causes immediate exit 1." Translate
		// to a single fatal aggregated into ProfileValidationError so
		// the cobra layer is uniform.
		out.Errors = append(out.Errors, profilevo.ValidationIssue{
			Code:    "path_not_found",
			Message: fmt.Sprintf("profile path %q does not exist", in.Path),
		})
		return out, &domerr.ProfileValidationError{
			Path:     abs,
			Errors:   []string{out.Errors[0].Message},
			Warnings: nil,
		}
	}
	if !info.IsDir() {
		msg := fmt.Sprintf("profile path %q is not a directory", in.Path)
		out.Errors = append(out.Errors, profilevo.ValidationIssue{Code: "not_a_directory", Message: msg})
		return out, &domerr.ProfileValidationError{
			Path: abs, Errors: []string{msg},
		}
	}

	// Step 2 — yaml.Node walk for unknown-classification-field warnings.
	// We do this BEFORE LoadBundle so that even when LoadBundle fails
	// (e.g., template missing) we still surface the warnings the user
	// can fix in the same edit cycle.
	warnings := scanClassificationKeyTypos(abs)
	for _, w := range warnings {
		out.Warnings = append(out.Warnings, profilevo.ValidationIssue{
			Code:    "unknown_classification_field",
			Message: w,
		})
	}

	// Step 3 — leverage LoadBundle for the structural fatals.
	bundle, loadErr := u.loader.LoadBundle(ctx, profilevo.LoadProfileBundleInput{Dir: abs})
	if loadErr != nil {
		out.Errors = append(out.Errors, profilevo.ValidationIssue{
			Code:    classifyLoadErr(loadErr),
			Message: humanizeLoadErr(loadErr),
		})
	}

	// Step 4a — explicit "missing name" check. We need the raw DTO for
	// this; the loader's bundle exposes Profile.Name only when LoadBundle
	// succeeded. When LoadBundle failed BEFORE name validation (e.g.
	// profile.yaml missing) we skip this check — the user has bigger
	// problems. When LoadBundle failed AFTER name was decoded (e.g.
	// missing template), we still inspect the raw bundleDTO via a tiny
	// inline yaml decode of the same `name:` key.
	if loadErr == nil || isAfterNameDecode(loadErr) {
		if rawName, _ := readRawNameField(abs); rawName == "" {
			out.Errors = append(out.Errors, profilevo.ValidationIssue{
				Code:    "missing_name",
				Message: "missing required field: name",
			})
		}
	}

	// Step 4b — duplicate type-id detection via raw yaml.Node scan
	// (independent of LoadBundle so we still detect duplicates when the
	// load fails for a separate reason). The duplicate scan walks
	// types[] in document order and reports the SECOND occurrence.
	for _, dup := range scanDuplicateTypeIDs(abs) {
		out.Errors = append(out.Errors, profilevo.ValidationIssue{
			Code:    "duplicate_type_id",
			Message: fmt.Sprintf("duplicate type id: %s", dup),
		})
	}

	// Step 5 — aggregate. _ = bundle (the use case does not use the
	// loaded aggregate beyond the load-error signal in this US).
	_ = bundle
	if len(out.Errors) == 0 {
		return out, nil
	}
	errMsgs := make([]string, 0, len(out.Errors))
	for _, e := range out.Errors {
		errMsgs = append(errMsgs, e.Message)
	}
	warnMsgs := make([]string, 0, len(out.Warnings))
	for _, w := range out.Warnings {
		warnMsgs = append(warnMsgs, w.Message)
	}
	sort.Strings(errMsgs) // deterministic order for stderr/test asserts
	return out, &domerr.ProfileValidationError{
		Path:     abs,
		Errors:   errMsgs,
		Warnings: warnMsgs,
	}
}
```

### 5.4 Helper functions (file-private)

- `scanClassificationKeyTypos(dir string) []string` — opens
  `<dir>/profile.yaml`, decodes into a `yaml.Node`, walks
  `types[].classification[]` mapping nodes, compares each key against
  the canonical set
  `{kind, implements_all, implements_none, has_annotation,
  path_contains}` (matches `bundleDto.go:89-95`), and returns formatted
  warning strings of the form `unknown classification field 'KEY'`.
  Single quotes around the key — pinned by the .feature line 223.
- `scanDuplicateTypeIDs(dir string) []string` — yaml.Node walk over
  `types[].id`; first occurrence registered, every later collision
  appended (so two collisions emit two warnings, not one).
- `readRawNameField(dir string) (string, error)` — minimal yaml.Node
  read of the top-level `name:` scalar.
- `classifyLoadErr(err error) string` — switch on `errors.Is` /
  `errors.As` returning a stable code (`yaml_missing`,
  `template_missing`, `language_unsupported`, `profile_invalid`).
- `humanizeLoadErr(err error) string` — for `*TemplateMissingError`,
  return `err.Error()` verbatim (already names the template file —
  satisfies .feature scenario 3); for `ErrProfileYamlMissing`, return
  `profile.yaml not found`; for the rest, return `err.Error()`.
- `isAfterNameDecode(err error) bool` — returns true when the load
  failure happened POST-decode of profile.yaml. Equivalent to "not
  ErrProfileYamlMissing and not a yaml decode error". Used to gate
  Step 4a so we don't double-report when profile.yaml is unreadable.

### 5.5 Error wrapping policy

- Every helper returns an unwrapped string (never an error) — fatals
  flow through `out.Errors` as plain `ValidationIssue` records and are
  re-wrapped once in `*ProfileValidationError` at the end of `Execute`.
- The bundle load error itself is preserved as the `Code` discriminator
  but its message is humanised (typed errors carry useful prefixes —
  `*TemplateMissingError` already self-formats with the missing
  filename, so we re-emit `err.Error()` verbatim).

### 5.6 Context handling

- `ctx.Err()` checked at entry only (the operation is single-shot,
  `os.Stat` and a handful of yaml decodes; no long loops).
- The yaml.Node walk does not poll context — the file is small.

---

## Section 6 — Presentation Layer Plan

### 6.1 `internal/cli/command/profileValidateCmd.go` (new — Group T4-G1)

```go
package command

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/jitctx/jitctx/internal/cli/format"
	"github.com/jitctx/jitctx/internal/domain/usecase/profilevalidateuc"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// NewProfileValidateCmd constructs the "profile validate <path>" cobra
// subcommand. Positional arg: path to a profile directory (required).
func NewProfileValidateCmd(uc profilevalidateuc.UseCase, _ *slog.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "validate <path>",
		Short: "Validate a profile directory for structural and logical errors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			out, err := uc.Execute(cmd.Context(), profilevo.ValidateProfileInput{Path: path})

			// Print warnings to stderr REGARDLESS of error state, so
			// scenario 4 ("warning, may still exit 0") is satisfied.
			for _, w := range out.Warnings {
				fmt.Fprintln(cmd.ErrOrStderr(), w.Message)
			}
			if err != nil {
				return format.TranslateError(err)
			}
			_, perr := fmt.Fprintln(cmd.OutOrStdout(), "Profile valid")
			return perr
		},
	}
}
```

Note: the warnings loop runs even on the success path (out.Warnings is
populated by the use case on both branches). On the error branch the
typed error renders through `format.TranslateError` to stderr — the
warnings printed first are independent.

### 6.2 `internal/cli/command/profileCmd.go` (modify)

Single-line edit: extend the `Long` field so help text mentions
`validate`:

```go
Long: `Manage framework profiles. Subcommands:

  init <name>         Extract a bundled profile into .jitctx/profiles/<name>/
  validate <path>     Validate a profile directory for structural errors`,
```

`Short` already says "Manage framework profiles (init, validate)" — no
change there. The actual `AddCommand` wiring lives in `root.go` (Tier 5,
Section 7.2).

### 6.3 `internal/cli/format/errors.go` (modify)

Insert a new `errors.As` branch BEFORE the catch-all `default` block at
the end of `TranslateError` (specifically after the
`*UnknownBundledProfileError` block, before
`if errors.Is(err, domerr.ErrProfileNotFound)`):

```go
var pve *domerr.ProfileValidationError
if errors.As(err, &pve) {
	var b strings.Builder
	fmt.Fprintf(&b, "profile %q: %d error(s)", pve.Path, len(pve.Errors))
	for _, msg := range pve.Errors {
		fmt.Fprintf(&b, "\n  - %s", msg)
	}
	return errors.New(b.String())
}
```

Rationale: cobra prints `errors.New(b.String())` to stderr verbatim and
the surrounding `SilenceUsage:true` keeps usage chatter out. Each fatal
becomes a bullet line — the `.feature` only asserts `stderr contains
<literal>`, so any rendering that puts the literal substring on stderr
is acceptable.

### 6.4 stdout/stderr contract

- **Success (no fatals)**: stdout = `Profile valid\n`. Stderr =
  zero or more warning lines. Exit 0.
- **Failure (one or more fatals)**: stdout = empty. Stderr = warning
  lines (if any) followed by the rendered ProfileValidationError block.
  Exit 1 (cobra's default for non-nil `RunE` error).

### 6.5 Exit codes

- 0 — clean or warnings-only.
- 1 — any fatal (including path-not-found, yaml missing, template
  missing, missing name, duplicate type id, language unsupported).
- 2 — bad usage (wrong arg count); cobra default. The .feature does
  not require special handling.

---

## Section 7 — Composition Root + Tests Plan

### 7.1 `internal/cli/wire.go` (modify — Group T5-G1)

Three small edits:

(a) Add the import:
```go
appprofilevalidateuc "github.com/jitctx/jitctx/internal/application/usecase/profilevalidateuc"
"github.com/jitctx/jitctx/internal/domain/usecase/profilevalidateuc"
```

(b) Add the field to `Deps`:
```go
// ValidateProfile is the profile-validate use case. Consumed by the
// new "profile validate" cobra subcommand. EP04US-007.
ValidateProfile profilevalidateuc.UseCase
```

(c) Construct and assign inside `Wire()`:
```go
validateProfileUC := appprofilevalidateuc.New(profileBundleLoader, logger)

// ... and in the returned Deps literal:
ValidateProfile: validateProfileUC,
```

### 7.2 `internal/cli/root.go` (modify — Group T5-G1)

Add a single line after the existing `AddCommand(NewProfileInitCmd(...))`
call inside `NewRootCmd`:

```go
profileCmd.AddCommand(command.NewProfileInitCmd(d.InitProfile, d.ProfilesDir, d.Logger))
profileCmd.AddCommand(command.NewProfileValidateCmd(d.ValidateProfile, d.Logger)) // NEW
```

### 7.3 Unit tests — Group T6-G1

`internal/application/usecase/profilevalidateuc/usecase_test.go`:

- **fakeLoadProfileBundlePort** — local fake that records the input dir
  and returns a configurable error/bundle pair (mirrors
  `fakeListBundledProfilesPort` in `profileinituc/usecase_test.go`).
- **TestValidateUC_CleanProfile** — fake returns a non-nil bundle, no
  warnings. Asserts: nil error, `len(out.Errors)==0`,
  `len(out.Warnings)==0`. Uses an inline `t.TempDir()` profile.yaml.
- **TestValidateUC_PathDoesNotExist** — Path points to a non-existent
  dir. Asserts: `errors.As(err, &*ProfileValidationError{})`,
  `errors.Is(err, ErrProfileValidationFailed)`, message contains
  `does not exist`. Loader is NOT called (verified via call counter on
  the fake).
- **TestValidateUC_MissingName** — `t.TempDir()` profile.yaml without
  `name:`, fake loader configured to return nil (so name is the only
  fatal). Asserts: error is `*ProfileValidationError`, `Errors`
  contains the literal `missing required field: name`.
- **TestValidateUC_MissingTemplate** — fake loader returns a
  `*TemplateMissingError`; the use case must surface it as a fatal
  whose message contains `references missing template`. Asserts the
  template basename appears in the rendered message.
- **TestValidateUC_UnknownClassificationField_IsWarning** —
  profile.yaml under `t.TempDir()` declares a classification entry
  with key `implementss`. Fake loader returns nil (the loader uses
  KnownFields(false) and ignores typos). Asserts: nil error
  (success path), `out.Warnings` contains the literal
  `unknown classification field 'implementss'` (single quotes).
- **TestValidateUC_DuplicateTypeIDs** — profile.yaml with two `id:
  service` types. Fake loader returns nil (the loader does not detect
  duplicates today). Asserts: error is `*ProfileValidationError`,
  message contains `duplicate type id: service`.
- **TestValidateUC_FatalsAreSorted** — multiple fatals; asserts the
  Errors slice order is alphabetic (so format/errors.go renders a
  stable bullet list). This is the contract that the integration
  test relies on.
- **TestValidateUC_WarningsPrintedEvenWhenFatalPresent** — combine
  scenarios 4 + 5 (typo + duplicate ids). Asserts: error is non-nil,
  Warnings slice is non-empty.

### 7.4 Integration test — Group T6-G2

`internal/cli/command/profileValidateIntegration_test.go`:

Wires a real `*fsprofile.BundleLoader` (with `nil` LanguageQueries —
matches existing test sites — see `bundleLoader.go:45-49`, nil-tolerant).
Five tests, each consuming a fixture under `testdata/ep04us007/`:

- **TestProfileValidateCmd_CleanProfileExitsZero** — fixture
  `cleanProfile/`. Asserts stdout contains `Profile valid`, no error,
  stderr empty.
- **TestProfileValidateCmd_MissingNameExitsOne** — fixture
  `missingName/`. Asserts error returned, error message contains
  `missing required field: name`.
- **TestProfileValidateCmd_MissingTemplateExitsOne** — fixture
  `missingTemplate/`. Asserts error returned, error message names
  `service.java.tmpl` (or whatever the fixture references).
- **TestProfileValidateCmd_UnknownClassificationFieldWarnsExitsZero**
  — fixture `unknownClassificationField/`. Asserts no error, stderr
  contains `unknown classification field 'implementss'`, stdout
  contains `Profile valid`.
- **TestProfileValidateCmd_DuplicateTypeIdsExitsOne** — fixture
  `duplicateTypeIds/`. Asserts error returned, message contains
  `duplicate type id: service`.

Helpers: `buildProfileValidateCmd(t)` mirrors
`buildProfileInitCmd` (factory wired from real infra adapters).

### 7.5 Fixtures — Group T6-G3

Five profile directories under `testdata/ep04us007/`:

- `cleanProfile/profile.yaml` — minimal valid: `name: clean`,
  `language: java`, one type with `id: service`,
  `template: service.java.tmpl`, classification using only known keys.
- `cleanProfile/templates/service.java.tmpl` — empty stub.
- `missingName/profile.yaml` — same as clean but with the `name:` line
  removed (or `name: ""`). NO templates dir needed (the `types: []`
  list is empty in this fixture so the loader does not fail on
  templates first).
- `missingTemplate/profile.yaml` — declares `id: service`,
  `template: service.java.tmpl`, but no `templates/` dir on disk.
- `unknownClassificationField/profile.yaml` — declares `id: service`,
  `template: service.java.tmpl`, classification entry
  `implementss: [Foo]` (typo).
- `unknownClassificationField/templates/service.java.tmpl` — empty
  stub (so the only issue is the typo, which is a WARNING — exit 0).
- `duplicateTypeIds/profile.yaml` — two `types[]` entries both with
  `id: service`, both with `template: service.java.tmpl`.
- `duplicateTypeIds/templates/service.java.tmpl` — empty stub.

All fixtures pin `language: java` so the registry resolution does not
fail (see Section 8 Q1).

---

## Section 8 — Open Questions & Risks

### Q1 — Should validate also surface unsupported languages? (resolved)

**Resolution:** Yes — for free. Production wiring already injects a
real `tsbundled.NewRegistry(logger)` into `BundleLoader`
(`wire.go:134-135`), so any profile.yaml declaring an unknown
`language:` value triggers `*LanguageUnsupportedError` inside
`LoadBundle` and the use case surfaces it as a fatal via
`humanizeLoadErr`. The .feature does not pin a scenario for this, but
covering it costs zero extra code and aligns with RNF-006. **Blocking:
No.**

### Q2 — How should unknown classification keys be detected? (resolved)

**Resolution:** yaml.Node walk over `types[].classification[]` mapping
nodes only. Rationale:
- A strict re-decode (`KnownFields(true)`) would warn on every
  forward-compatible field the runtime ignores (e.g., a future
  `packaging:` extension). False positives swamp the report.
- Walking only the classification subtree mirrors what the runtime
  actually consumes (`bundleDto.go:89-95`), and the canonical key set
  is small and stable.
- This pulls `gopkg.in/yaml.v3` into the application layer — a
  documented exception. The alternative (publish a domain port +
  fsprofile adapter) duplicates the canonical key set in
  infrastructure code where it does not belong.

A file-level comment in `usecase.go` documents the import exception so
future reviewers don't flag it as a guideline violation. **Blocking:
No.**

### Q3 — Where does the literal `Profile valid` live? (resolved)

**Resolution:** in the cobra command (`profileValidateCmd.go`). The
use case is a domain-friendly success/error split; the human-friendly
success line is presentation. **Blocking: No.**

### Q4 — Positional arg vs `--path` flag? (resolved)

**Resolution:** positional arg with `cobra.ExactArgs(1)`. The .feature
pins `jitctx profile validate <path>`. **Blocking: No.**

### Q5 — Risk: what if profile.yaml is malformed YAML at the byte level?

**Mitigation:** `LoadBundle` already returns a wrapped
`fmt.Errorf("decode profile.yaml: %w", domerr.ErrProfileInvalid)`
(`bundleLoader.go:165`). The use case surfaces this as a fatal via
`humanizeLoadErr`. The yaml.Node helpers
(`scanClassificationKeyTypos`, `scanDuplicateTypeIDs`,
`readRawNameField`) MUST tolerate decode failures silently — they
return empty results when the file cannot be parsed, deferring to the
LoadBundle fatal for the user-visible error. **Blocking: No.**

### Q6 — Risk: command-help string drift

The `Long` text on `profileCmd.go` was authored in EP04US-006 and
mentions only `init`. Updating it in this US is a one-line edit in
the same file as the cobra parent. No risk. **Blocking: No.**

---

## Section 9 — Parallel Execution Plan (authoritative for @agent-manager)

```yaml
tiers:
  - id: 1
    name: Domain contract
    depends_on: []
    groups:
      - id: T1-G1
        scope:
          create:
            - internal/domain/usecase/profilevalidateuc/port.go
            - internal/domain/vo/profile/validateProfileInput.go
            - internal/domain/vo/profile/validateProfileOutput.go
          modify:
            - internal/domain/errors/errors.go
        guidelines:
          - .claude/guidelines/domain-layer-guidelines.yml
        effort: S
        notes: >
          Publishes the contract (UseCase interface, two VOs, sentinel
          + typed error) consumed by Tier 3 (use case impl) and Tier 4
          (cobra cmd + format/errors translator). Append-only edit to
          internal/domain/errors/errors.go — do not touch existing
          declarations.

  - id: 3
    name: Application use case
    depends_on: [1]
    groups:
      - id: T3-G1
        scope:
          create:
            - internal/application/usecase/profilevalidateuc/usecase.go
        guidelines:
          - .claude/guidelines/application-layer-guidelines.yml
        effort: M
        notes: >
          Imports gopkg.in/yaml.v3 by exception (documented in a
          file-level comment) — see Section 8 Q2. Consumes the
          existing profileport.LoadProfileBundlePort. Delegates
          structural fatals to LoadBundle and adds three explicit
          checks (missing name, duplicate type ids, unknown
          classification keys via yaml.Node walk).

  - id: 4
    name: Presentation (cobra cmd + format/errors)
    depends_on: [1, 3]
    groups:
      - id: T4-G1
        scope:
          create:
            - internal/cli/command/profileValidateCmd.go
          modify:
            - internal/cli/command/profileCmd.go
            - internal/cli/format/errors.go
        guidelines:
          - .claude/guidelines/presentation-layer-guidelines.yml
        effort: S
        notes: >
          New cobra subcommand with cobra.ExactArgs(1). Updates the
          profileCmd Long help text. Adds an errors.As branch in
          format/errors.go for *ProfileValidationError that renders
          fatals as bullet lines on stderr. Warnings stream from the
          cobra RunE BEFORE TranslateError so they print regardless
          of exit state.

  - id: 5
    name: Composition root
    depends_on: [3, 4]
    groups:
      - id: T5-G1
        scope:
          modify:
            - internal/cli/wire.go
            - internal/cli/root.go
        guidelines:
          - .claude/guidelines/main-layer-guidelines.yml
        effort: S
        notes: >
          Adds Deps.ValidateProfile (one new field), constructs
          appprofilevalidateuc.New(profileBundleLoader, logger), wires
          it into root.go via profileCmd.AddCommand alongside the
          existing init subcommand.

  - id: 6
    name: Tests + fixtures
    depends_on: [5]
    groups:
      - id: T6-G1
        scope:
          create:
            - internal/application/usecase/profilevalidateuc/usecase_test.go
        guidelines:
          - .claude/guidelines/unit-test-layer-guidelines.yml
        effort: M
        notes: >
          Eight unit tests covering all five .feature scenarios plus
          path-not-found, sort-order, and warning-with-fatal combos.
          Uses a hand-rolled fakeLoadProfileBundlePort mirroring the
          profileinituc test pattern. Profile YAML fixtures are
          inlined as t.TempDir() writes so the unit test has zero
          testdata coupling.

      - id: T6-G2
        scope:
          create:
            - internal/cli/command/profileValidateIntegration_test.go
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: M
        notes: >
          Five end-to-end tests, one per .feature scenario, wired with
          real *fsprofile.BundleLoader (nil LanguageQueries — matches
          existing test sites). buildProfileValidateCmd helper mirrors
          buildProfileInitCmd. Asserts on stdout/stderr literals
          pinned by the .feature.

      - id: T6-G3
        scope:
          create:
            - testdata/ep04us007/cleanProfile/profile.yaml
            - testdata/ep04us007/cleanProfile/templates/service.java.tmpl
            - testdata/ep04us007/missingName/profile.yaml
            - testdata/ep04us007/missingTemplate/profile.yaml
            - testdata/ep04us007/unknownClassificationField/profile.yaml
            - testdata/ep04us007/unknownClassificationField/templates/service.java.tmpl
            - testdata/ep04us007/duplicateTypeIds/profile.yaml
            - testdata/ep04us007/duplicateTypeIds/templates/service.java.tmpl
        guidelines:
          - .claude/guidelines/integration-test-layer-guidelines.yml
        effort: S
        notes: >
          Five profile-directory fixtures, one per .feature scenario.
          Templates dirs include a stub service.java.tmpl whenever the
          fixture is meant to PASS the missing-template check (so only
          missingTemplate/ omits the templates dir). All YAMLs pin
          language: java to satisfy the production-wired registry.
```
