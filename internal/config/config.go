package config

type Config struct {
	WorkDir      string
	ManifestPath string
	// ProfilesDir is the sole source of framework profiles. The jitctx binary
	// reads profile YAML files from this directory at runtime; there is no
	// bundled fallback. Users copy sample profiles from the repo-root
	// profiles/ directory into their project's .jitctx/profiles/ directory.
	ProfilesDir   string
	DefaultBudget int
	LogLevel      string

	// PlansDir is the directory where `jitctx plan --new` writes new
	// spec templates and where future read-side resolution will look
	// for specs. When empty, the convention "./jitctx-plans" applies.
	// Loaded from .jitctx/config.yaml key "plans_dir" (write-side use
	// for EP02US-002; read-side resolution arrives with EP02US-007).
	PlansDir string
}

func Defaults() Config {
	return Config{
		WorkDir:       ".",
		ManifestPath:  "project-state.yaml",
		ProfilesDir:   ".jitctx/profiles",
		DefaultBudget: 0,
		LogLevel:      "info",
	}
}
