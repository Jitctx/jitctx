package fsmanifest

import (
	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/vo"
)

func toDomain(d projectStateDTO) *model.ProjectState {
	state := &model.ProjectState{
		GeneratedAt: d.GeneratedAt,
		Stack: model.Stack{
			Languages:  d.Stack.Languages,
			Frameworks: d.Stack.Frameworks,
		},
	}
	for _, m := range d.Modules {
		state.Modules = append(state.Modules, model.Module{
			ID:           m.ID,
			Path:         m.Path,
			Tags:         m.Tags,
			Contracts:    mapContractsToDomain(m.Contracts),
			Dependencies: m.Dependencies,
		})
	}
	for _, c := range d.Contexts {
		state.Contexts = append(state.Contexts, model.Context{
			ID:            c.ID,
			Type:          vo.ArtifactType(c.Type),
			AppliesTo:     c.AppliesTo,
			Module:        c.Module,
			Tags:          c.Tags,
			Path:          c.Path,
			TokenEstimate: c.TokenEstimate,
		})
	}
	return state
}

func toDTO(s *model.ProjectState) projectStateDTO {
	dto := projectStateDTO{
		GeneratedAt: s.GeneratedAt,
		Stack: stackDTO{
			Languages:  s.Stack.Languages,
			Frameworks: s.Stack.Frameworks,
		},
	}
	for _, m := range s.Modules {
		tags := m.Tags
		if tags == nil {
			tags = []string{}
		}
		deps := m.Dependencies
		if deps == nil {
			deps = []string{}
		}
		dto.Modules = append(dto.Modules, moduleDTO{
			ID:           m.ID,
			Path:         m.Path,
			Tags:         tags,
			Contracts:    mapContractsToDTO(m.Contracts),
			Dependencies: deps,
		})
	}
	for _, c := range s.Contexts {
		tags := c.Tags
		if tags == nil {
			tags = []string{}
		}
		dto.Contexts = append(dto.Contexts, contextDTO{
			ID:            c.ID,
			Type:          string(c.Type),
			AppliesTo:     c.AppliesTo,
			Module:        c.Module,
			Tags:          tags,
			Path:          c.Path,
			TokenEstimate: c.TokenEstimate,
		})
	}
	return dto
}

func mapContractsToDomain(in []contractDTO) []model.Contract {
	if len(in) == 0 {
		return nil
	}
	out := make([]model.Contract, 0, len(in))
	for _, c := range in {
		methods := make([]model.Method, 0, len(c.Methods))
		for _, m := range c.Methods {
			methods = append(methods, model.Method{Signature: m.Signature})
		}
		out = append(out, model.Contract{
			Name:    c.Name,
			Type:    model.ContractType(c.Type),
			Path:    c.Path,
			Methods: methods,
		})
	}
	return out
}

func mapContractsToDTO(in []model.Contract) []contractDTO {
	if len(in) == 0 {
		return []contractDTO{}
	}
	out := make([]contractDTO, 0, len(in))
	for _, c := range in {
		methods := make([]methodDTO, 0, len(c.Methods))
		for _, m := range c.Methods {
			methods = append(methods, methodDTO{Signature: m.Signature})
		}
		out = append(out, contractDTO{
			Name:    c.Name,
			Type:    string(c.Type),
			Path:    c.Path,
			Methods: methods,
		})
	}
	return out
}
