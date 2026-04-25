package service

import (
	"sort"
	"strings"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
	planvo "github.com/jitctx/jitctx/internal/domain/vo/plan"
)

// DependencyLayerer performs Kahn-style topological sorting over a set
// of SpecContract dependencies and groups the result into parallel
// execution layers. Pure (no I/O); deterministic alphabetical ordering
// within each layer; reports cycles as *CycleError.
//
// Behaviour guarantees:
//   - Layer 0 contains every contract whose Uses/Implements/DependsOn
//     references resolve only to externals (i.e. no in-spec dependency).
//   - A contract appears in layer N when all of its in-spec dependencies
//     have appeared in layer < N.
//   - Externals (names not present in the spec) are treated as
//     pre-satisfied and DO NOT contribute to any layer.
//   - Cycle detection runs after externals are excluded; the returned
//     CycleError.Path is a deterministic representative cycle (alphabet-
//     ically smallest start node, traversal in dependency-order).
type DependencyLayerer struct{}

// NewDependencyLayerer returns a stateless layerer.
func NewDependencyLayerer() DependencyLayerer { return DependencyLayerer{} }

// Layer computes layers for the given contracts. The returned externals
// slice is the sorted union of all unresolved Uses/Implements/DependsOn
// names across every contract. PlanTarget.TargetPath is left empty by
// this service; the use case fills it via ContractPathMapper.
func (DependencyLayerer) Layer(contracts []model.SpecContract) ([]planvo.ExecutionLayer, []string, error) {
	if len(contracts) == 0 {
		return nil, nil, nil
	}

	// Step 1: build name→contract map and inSpec set.
	contractMap := make(map[string]model.SpecContract, len(contracts))
	inSpec := make(map[string]bool, len(contracts))
	for _, c := range contracts {
		contractMap[c.Name] = c
		inSpec[c.Name] = true
	}

	// Step 2: for each contract c, deps[c.Name] = sorted distinct intersection
	// of inSpec with (Uses ∪ {Implements} ∪ DependsOn). Skip empty Implements.
	deps := make(map[string][]string, len(contracts))
	// Step 3: externals = sorted distinct (union of all refs minus inSpec).
	externalSet := make(map[string]bool)

	for _, c := range contracts {
		allRefs := collectRefs(c)
		var inSpecDeps []string
		seen := make(map[string]bool)
		for _, ref := range allRefs {
			if inSpec[ref] {
				if !seen[ref] {
					seen[ref] = true
					inSpecDeps = append(inSpecDeps, ref)
				}
			} else {
				externalSet[ref] = true
			}
		}
		sort.Strings(inSpecDeps)
		deps[c.Name] = inSpecDeps
	}

	externals := sortedKeys(externalSet)

	// Step 4: Kahn's algorithm — repeatedly emit one layer of contracts
	// whose deps ⊆ emitted; sorted alphabetically within each layer.
	emitted := make(map[string]bool)
	remaining := make(map[string]bool, len(contracts))
	for _, c := range contracts {
		remaining[c.Name] = true
	}

	var layers []planvo.ExecutionLayer
	layerIdx := 0

	for len(remaining) > 0 {
		// Find all contracts whose in-spec deps are all emitted.
		var ready []string
		for name := range remaining {
			if allSatisfied(deps[name], emitted) {
				ready = append(ready, name)
			}
		}

		if len(ready) == 0 {
			// Cycle detected — find the alphabetically smallest unresolved node
			// and DFS to extract the cycle path.
			cycleErr := findCycle(remaining, deps)
			return nil, nil, cycleErr
		}

		sort.Strings(ready)

		targets := make([]planvo.PlanTarget, 0, len(ready))
		for _, name := range ready {
			c := contractMap[name]
			targets = append(targets, planvo.PlanTarget{
				Name:       c.Name,
				Type:       string(c.Type),
				Uses:       c.Uses,
				Implements: c.Implements,
				DependsOn:  c.DependsOn,
			})
			emitted[name] = true
			delete(remaining, name)
		}

		layers = append(layers, planvo.ExecutionLayer{
			Index:   layerIdx,
			Targets: targets,
		})
		layerIdx++
	}

	return layers, externals, nil
}

// collectRefs returns all dependency references for a contract
// (Uses ∪ {Implements} ∪ DependsOn).
func collectRefs(c model.SpecContract) []string {
	var refs []string
	refs = append(refs, c.Uses...)
	if c.Implements != "" {
		refs = append(refs, c.Implements)
	}
	refs = append(refs, c.DependsOn...)
	return refs
}

// allSatisfied returns true if every dep in deps is in the emitted set.
func allSatisfied(deps []string, emitted map[string]bool) bool {
	for _, d := range deps {
		if !emitted[d] {
			return false
		}
	}
	return true
}

// sortedKeys returns a sorted slice of map keys.
func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// findCycle finds a cycle among the remaining unresolved nodes using DFS.
// It starts from the alphabetically smallest node for determinism.
func findCycle(remaining map[string]bool, deps map[string][]string) *CycleError {
	// Sort remaining names for deterministic start.
	names := sortedKeys(remaining)
	start := names[0]

	// DFS from start, only traversing edges within 'remaining'.
	path := []string{}
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var dfs func(node string) []string
	dfs = func(node string) []string {
		if inStack[node] {
			// Found back-edge — extract the cycle.
			idx := -1
			for i, n := range path {
				if n == node {
					idx = i
					break
				}
			}
			cycle := append([]string(nil), path[idx:]...)
			cycle = append(cycle, node)
			return cycle
		}
		if visited[node] {
			return nil
		}
		visited[node] = true
		inStack[node] = true
		path = append(path, node)

		// Only traverse edges within remaining.
		for _, dep := range deps[node] {
			if remaining[dep] {
				if cycle := dfs(dep); cycle != nil {
					return cycle
				}
			}
		}

		path = path[:len(path)-1]
		inStack[node] = false
		return nil
	}

	if cycle := dfs(start); cycle != nil {
		return &CycleError{Path: cycle}
	}

	// If start didn't yield a cycle (shouldn't happen when remaining is non-empty
	// and no progress was possible), try each remaining node.
	for _, name := range names[1:] {
		if !visited[name] {
			if cycle := dfs(name); cycle != nil {
				return &CycleError{Path: cycle}
			}
		}
	}

	// Fallback: return a minimal cycle with just the first node.
	return &CycleError{Path: []string{start, start}}
}

// CycleError carries the deterministic representative cycle path. The
// error message is "dependency cycle detected: A -> B -> A" so the
// presentation layer can echo it verbatim per the .feature acceptance.
type CycleError struct {
	Path []string // e.g. ["A", "B", "A"]
}

func (e *CycleError) Error() string {
	return "dependency cycle detected: " + strings.Join(e.Path, " -> ")
}

// Is routes errors.Is(err, ErrDependencyCycle) to this error type.
func (e *CycleError) Is(target error) bool {
	return target == domerr.ErrDependencyCycle
}
