package fsconfig

import "github.com/jitctx/jitctx/internal/domain/model"

// toDomain maps the YAML DTO into the immutable domain JitctxConfig.
// Nil/empty `disabled_rules` collapses to a nil slice — equivalent to "no
// disabled rules" downstream.
func toDomain(dto configFileDTO) model.JitctxConfig {
	var disabled []string
	if len(dto.Audit.DisabledRules) > 0 {
		disabled = make([]string, 0, len(dto.Audit.DisabledRules))
		for _, id := range dto.Audit.DisabledRules {
			if id == "" {
				continue
			}
			disabled = append(disabled, id)
		}
	}
	return model.JitctxConfig{
		Audit: model.JitctxAuditConfig{DisabledRules: disabled},
	}
}
