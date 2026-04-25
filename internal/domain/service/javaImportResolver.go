package service

import (
	"sort"
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// JavaImportResolver computes the alphabetically-sorted, deduplicated list
// of fully-qualified Java imports needed to render `target` given the rest
// of the spec.
//
// Algorithm (stateless):
//
//  1. For service/rest-adapter/jpa-adapter: for each name in target.DependsOn
//     (and target.Uses for rest-adapter), look up the contract by name in
//     spec.Contracts; if found, FQN = modulePackage + "." + dotPath(typePath).
//  2. For service: also FQN of target.Implements.
//  3. Append framework imports based on target.Type:
//     service        → "org.springframework.stereotype.Service"
//     rest-adapter   → "org.springframework.web.bind.annotation.RestController"
//     + per-endpoint VERB-mapping import (GetMapping, …)
//     entity / aggregate-root → "jakarta.persistence.Entity"
//     jpa-adapter    → "org.springframework.stereotype.Repository"
//  4. External names (not in spec) are ignored — the use case logs them as
//     warnings; the resolver does NOT fabricate FQNs.
//  5. Deduplicate, sort lexicographically.
//
// `mapper` is reused for typePath ("port/in/Foo.java" → "port.in"); the
// resolver depends on the Map() method, not the struct type, so a fake can
// be substituted in tests if needed.
type JavaImportResolver struct {
	mapper ContractPathMapper
}

// NewJavaImportResolver returns a JavaImportResolver backed by the given mapper.
func NewJavaImportResolver(mapper ContractPathMapper) JavaImportResolver {
	return JavaImportResolver{mapper: mapper}
}

// Resolve computes the sorted, deduplicated list of FQN imports for target.
// It never returns an error; unknown contract refs are silently skipped (the
// use case logs them separately).
func (r JavaImportResolver) Resolve(spec model.FeatureSpec, target model.SpecContract, modulePackage string) ([]string, error) {
	// Build name → SpecContract index.
	index := make(map[string]model.SpecContract, len(spec.Contracts))
	for _, c := range spec.Contracts {
		index[c.Name] = c
	}

	// Collect candidate ref names based on target.Type.
	var refs []string
	switch target.Type {
	case model.ContractService:
		refs = append(refs, target.Implements)
		refs = append(refs, target.DependsOn...)
	case model.ContractRestAdapter:
		refs = append(refs, target.Uses...)
		refs = append(refs, target.DependsOn...)
	case model.ContractJPAAdapter:
		refs = append(refs, target.Implements)
		refs = append(refs, target.DependsOn...)
	}

	seen := make(map[string]struct{})
	idUtils := NewJavaIdentifierUtils()

	// Resolve each ref to an FQN import.
	for _, name := range refs {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// Skip self-reference.
		if name == target.Name {
			continue
		}
		c, ok := index[name]
		if !ok {
			// External reference — skip silently.
			continue
		}
		relPath, err := r.mapper.Map(c.Type, c.Name)
		if err != nil {
			// Unsupported contract type — skip; the use case surfaces
			// UnsupportedContractTypeError separately when processing that contract.
			continue
		}
		fqn := idUtils.FQN(modulePackage, relPath, c.Name)
		seen[fqn] = struct{}{}
	}

	// Add framework imports based on target.Type.
	switch target.Type {
	case model.ContractService:
		seen["org.springframework.stereotype.Service"] = struct{}{}
	case model.ContractRestAdapter:
		seen["org.springframework.web.bind.annotation.RestController"] = struct{}{}
		// Per-endpoint verb-mapping imports.
		synth := NewEndpointSynthesizer()
		verbsSeen := make(map[string]struct{})
		for _, raw := range target.Endpoints {
			eb, err := synth.Parse(raw)
			if err != nil {
				continue
			}
			if _, already := verbsSeen[eb.Verb]; already {
				continue
			}
			verbsSeen[eb.Verb] = struct{}{}
			// e.g. "GET" → "GetMapping" → import path
			annotationName := strings.ToUpper(eb.Verb[:1]) + strings.ToLower(eb.Verb[1:]) + "Mapping"
			seen["org.springframework.web.bind.annotation."+annotationName] = struct{}{}
		}
	case model.ContractEntity:
		seen["jakarta.persistence.Entity"] = struct{}{}
	case model.ContractAggregate:
		seen["jakarta.persistence.Entity"] = struct{}{}
	case model.ContractJPAAdapter:
		seen["org.springframework.stereotype.Repository"] = struct{}{}
	}

	// Collect, sort, return.
	result := make([]string, 0, len(seen))
	for fqn := range seen {
		result = append(result, fqn)
	}
	sort.Strings(result)
	return result, nil
}
