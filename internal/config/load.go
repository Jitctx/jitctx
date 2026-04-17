package config

import "os"

func Load() (Config, error) {
	cfg := Defaults()
	if v := os.Getenv("JITCTX_WORKDIR"); v != "" {
		cfg.WorkDir = v
	}
	if v := os.Getenv("JITCTX_MANIFEST"); v != "" {
		cfg.ManifestPath = v
	}
	if v := os.Getenv("JITCTX_PROFILES_DIR"); v != "" {
		cfg.ProfilesDir = v
	}
	if v := os.Getenv("JITCTX_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	return cfg, nil
}
