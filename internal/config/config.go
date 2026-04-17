package config

type Config struct {
	WorkDir       string
	ManifestPath  string
	ProfilesDir   string
	DefaultBudget int
	LogLevel      string
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
