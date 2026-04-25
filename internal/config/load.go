package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type configFileDTO struct {
	PlansDir string `yaml:"plans_dir"`
}

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

	cfgFilePath := filepath.Join(cfg.WorkDir, ".jitctx", "config.yaml")
	f, err := os.Open(cfgFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return Config{}, fmt.Errorf("read .jitctx/config.yaml: %w", err)
		}
	} else {
		defer f.Close()
		dec := yaml.NewDecoder(f)
		dec.KnownFields(true)
		var dto configFileDTO
		if err := dec.Decode(&dto); err != nil {
			// An empty file (comment-only or zero bytes) returns io.EOF from
			// Decode. Treat it as "no overrides" and fall through to env vars.
			if !errors.Is(err, io.EOF) {
				return Config{}, fmt.Errorf("read .jitctx/config.yaml: %w", err)
			}
		}
		if dto.PlansDir != "" {
			cfg.PlansDir = dto.PlansDir
		}
	}

	if v := os.Getenv("JITCTX_PLANS_DIR"); v != "" {
		cfg.PlansDir = v
	}

	return cfg, nil
}
