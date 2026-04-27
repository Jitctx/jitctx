// Package qualitygate implements the Java/Spring identifier gate for the
// jitctx CLI (EP04US-009 / EP04RNF-002). It contains no production logic —
// only the data consumed by the gate test in javaReferencesGate_test.go.
package qualitygate

// ForbiddenTokens is the literal list of Java/Spring identifiers pinned by
// EP04RNF-002 (.feature line 261). No production Go file under internal/ or
// cmd/ may contain any of these tokens outside a Go comment or an explicitly
// named ExemptFiles entry. Order is stable for deterministic test failure
// messages.
var ForbiddenTokens = []string{
	"@Entity",
	"@RestController",
	"port/in",
	"application/",
	"JUnit",
	"Mockito",
	"javax",
	"spring",
}

// ExemptFiles lists repo-relative paths whose content is permitted to contain
// one or more ForbiddenTokens entries. Each entry carries an inline comment
// that names the requirement or future-cleanup story that justifies the
// exemption.
//
// Rules for maintainers:
//   - Adding a new entry requires a code-review justification referencing the
//     US/EP that authorises the Java literal.
//   - When a file in this list is cleaned up so that it no longer contains any
//     ForbiddenTokens entry, the companion test
//     TestJavaReferencesGate_HonoursExemptions will fail until the stale entry
//     is removed here. That failure is intentional — it enforces hygiene.
var ExemptFiles = []string{
	// The token-registry file itself: the ForbiddenTokens slice elements ARE
	// the forbidden tokens, by structural necessity. The companion test
	// TestJavaReferencesGate_HonoursExemptions guarantees this exemption is
	// never stale — the file always contains all eight tokens.
	"internal/qualitygate/exemptions.go",

	// EP-03 leftovers — Java-specific domain services whose literals must be
	// data-driven from the profile bundle in a future EP-04 follow-up. Tracked
	// under the post-EP-04 backlog item "scaffold imports/annotations from
	// profile". Until then, the string literals are the only practical
	// representation of the Spring/JPA layout convention.
	"internal/domain/service/javaImportResolver.go",
	"internal/domain/service/contractPathMapper.go",
	"internal/domain/service/testPathMapper.go",

	// EP04US-009 Tier A (Option C): consolidated application-layer Java
	// constants. Replaces the previously inline literals in
	// scaffolduc/usecase.go. When a future story makes scaffold imports and
	// annotations data-driven from the loaded ProfileBundle, this file is
	// deleted and this exemption entry is removed in the same PR.
	"internal/application/usecase/scaffolduc/javaScaffoldConstants.go",
}
