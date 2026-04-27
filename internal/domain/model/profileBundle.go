package model

// ProfileBundle is the in-memory aggregate produced by
// profile.LoadProfileBundlePort. It wraps the EP-03 FrameworkProfile
// (preserved verbatim during EP04US-001) and adds the directory-shape
// concerns introduced by EP04RF-001: eagerly-loaded template bytes,
// raw declarative-types data forwarded for US-002, and the raw packaging
// block forwarded for US-008. Both Raw* fields are intentionally
// untyped/unvalidated in this US; subsequent USes will land their
// schemas without re-reading the YAML.
type ProfileBundle struct {
	// Profile is the classic EP-03 FrameworkProfile populated from the
	// metadata + rules + audit_rules sections. Its Source is set to
	// ProfileSourceCustom or ProfileSourceBundled by the loader.
	Profile *FrameworkProfile

	// Dir is the absolute directory the bundle was loaded from. Empty
	// when Source == ProfileSourceBundled.
	Dir string

	// Templates is keyed by file basename relative to <Dir>/templates/.
	// Values are the raw bytes; renderers parse them lazily.
	Templates map[string][]byte

	// RawTypes carries the profile.yaml `types:` section verbatim as a
	// slice of opaque declarations. EP04US-002 introduces the validation
	// engine that consumes this; this US only records that the section
	// was parsed without inspecting field semantics beyond `id` and
	// `template`.
	RawTypes []ProfileTypeDeclaration

	// RawPackaging holds the profile.yaml `packaging:` block as raw YAML
	// bytes, or nil when the block is absent. EP04US-008 will introduce
	// the DSL evaluator that consumes these bytes.
	RawPackaging []byte

	// RawAuditRules carries the profile.yaml `audit_rules:` section
	// verbatim as the same []AuditRule that LoadAuditRulesPort would
	// have produced. Empty slice when the bundle has no audit_rules:
	// key. EP04US-004 introduces this field; the load-time mapper
	// (bundleMapper.toBundleDomain) populates it.
	RawAuditRules []AuditRule

	// LanguageQueries is the bundled Tree-sitter query set the loader
	// resolved from Profile.Language at load time. Nil when the profile
	// did not declare a language (legacy schema). When non-nil, the
	// pointer is shared across every ProfileBundle whose Profile.Language
	// matches — the registry caches by language id (EP04US-005 Scenario 3).
	LanguageQueries *LanguageQuerySet
}

// GetTemplate returns the bytes of the named template, keyed by
// basename relative to templates/. Returns (nil, false) when the
// template was not present at load time.
func (b *ProfileBundle) GetTemplate(name string) ([]byte, bool) {
	if b == nil || b.Templates == nil {
		return nil, false
	}
	v, ok := b.Templates[name]
	return v, ok
}
