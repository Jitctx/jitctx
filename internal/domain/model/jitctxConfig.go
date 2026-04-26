package model

// JitctxConfig is the project-scoped runtime configuration loaded from
// `<workDir>/.jitctx/config.yaml`. The struct mirrors only the keys
// EP03US-005 understands; unknown TOP-LEVEL keys produce a YAML decode
// error (KnownFields(true) in the loader).
//
// Zero value is meaningful: an absent config file (os.IsNotExist) yields
// a zero JitctxConfig{} which the use case treats as "no overrides".
//
// No struct tags here — YAML mapping lives in the infrastructure DTO
// (internal/infrastructure/fsconfig/dto.go). This keeps the domain free
// of marshalling concerns per the domain-layer guideline.
type JitctxConfig struct {
	Audit JitctxAuditConfig
}

// JitctxAuditConfig holds the audit-scoped knobs. EP03US-005 introduces
// only DisabledRules. New keys are additive.
type JitctxAuditConfig struct {
	// DisabledRules is the list of audit rule IDs the developer has
	// chosen to skip. Order is irrelevant; the filter dedupes and sorts
	// before use. nil and empty are equivalent ("no disabled rules").
	DisabledRules []string
}
