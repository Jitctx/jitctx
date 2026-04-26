package fsconfig

// configFileDTO is the YAML deserialization shape for
// `<workDir>/.jitctx/config.yaml` as understood by EP03US-005. Other
// top-level keys live in the binary-launch loader (internal/config/load.go)
// — this DTO models ONLY the audit view because the use case loads the
// file independently per workdir.
//
// IMPORTANT: KnownFields(true) is used in loader.go to reject unknown
// top-level keys. To stay forward-compatible with the binary-launch
// loader (which already understands "plans_dir"), `PlansDir` is included
// here as an inert field so the strict decode does not reject configs
// that already exist for EP02 users.
type configFileDTO struct {
	PlansDir string         `yaml:"plans_dir"`
	Audit    auditConfigDTO `yaml:"audit"`
}

// auditConfigDTO models the `audit:` block. Only `disabled_rules` is
// recognised today (EP03RF-008 scope). Adding new keys is intentionally
// out of scope for this US.
type auditConfigDTO struct {
	DisabledRules []string `yaml:"disabled_rules"`
}
