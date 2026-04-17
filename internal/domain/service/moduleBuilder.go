package service

import (
	"path/filepath"
	"strings"

	"github.com/jitctx/jitctx/internal/domain/model"
)

// reservedHexagonalSegments are path segments that are part of the hexagonal
// structure and should be skipped when deriving the module root directory.
var reservedHexagonalSegments = map[string]bool{
	"port":           true,
	"in":             true,
	"out":            true,
	"adapter":        true,
	"domain":         true,
	"application":    true,
	"service":        true,
	"infrastructure": true,
	"repository":     true,
	"dto":            true,
}

// groupDepth is the number of initial path segments under src/main/java/
// to skip (e.g., com/app → depth=2).
const groupDepth = 2

// BuildModules groups JavaFileSummary declarations into modules using the hexagonal strategy.
func BuildModules(
	summaries []model.JavaFileSummary,
	prof *model.FrameworkProfile,
) []model.Module {
	moduleMap := make(map[string]*model.Module)

	for _, summary := range summaries {
		for _, decl := range summary.Declarations {
			contractType, ok := ClassifyDeclaration(decl, summary.Path, prof)
			if !ok {
				continue
			}

			moduleRoot, moduleID := deriveModuleRoot(summary.Path)
			if moduleID == "" {
				continue
			}

			if _, exists := moduleMap[moduleID]; !exists {
				moduleMap[moduleID] = &model.Module{
					ID:   moduleID,
					Path: moduleRoot,
					Tags: []string{},
				}
			}

			methods := make([]model.Method, 0, len(decl.Methods))
			for _, m := range decl.Methods {
				methods = append(methods, model.Method{Signature: m.Signature})
			}

			moduleMap[moduleID].Contracts = append(moduleMap[moduleID].Contracts, model.Contract{
				Name:    decl.Name,
				Type:    contractType,
				Path:    summary.Path,
				Methods: methods,
			})
		}
	}

	modules := make([]model.Module, 0, len(moduleMap))
	for _, m := range moduleMap {
		modules = append(modules, *m)
	}
	return modules
}

// deriveModuleRoot computes the module root directory path and module ID for a file.
// It finds the first non-reserved segment after src/main/java/<group>/<artifact>/.
func deriveModuleRoot(filePath string) (rootPath, moduleID string) {
	filePath = filepath.ToSlash(filePath)

	// Find src/main/java prefix.
	const javaRoot = "src/main/java/"
	_, afterJavaRoot, found := strings.Cut(filePath, javaRoot)
	if !found {
		return "", ""
	}

	segments := strings.Split(strings.TrimSuffix(afterJavaRoot, "/"), "/")
	// Remove the filename from segments.
	if len(segments) > 0 && strings.Contains(segments[len(segments)-1], ".") {
		segments = segments[:len(segments)-1]
	}

	// Skip groupDepth segments (e.g., com/app).
	if len(segments) <= groupDepth {
		return "", ""
	}
	segments = segments[groupDepth:]

	// The first segment that is NOT a reserved hexagonal segment is the module root.
	moduleSegmentIdx := -1
	for i, seg := range segments {
		if !reservedHexagonalSegments[seg] {
			moduleSegmentIdx = i
			break
		}
	}
	if moduleSegmentIdx < 0 {
		return "", ""
	}

	moduleSeg := segments[moduleSegmentIdx]
	moduleID = strings.ReplaceAll(strings.ToLower(moduleSeg), "_", "-")

	// Compute the module root path relative to the project root.
	allSegments := strings.Split(afterJavaRoot, "/")
	// Rebuild: java root segments + groupDepth + moduleSegment.
	rootParts := allSegments[:groupDepth+moduleSegmentIdx+1]
	rootPath = javaRoot[:len(javaRoot)-1] // no trailing slash
	if len(rootParts) > 0 {
		rootPath = filepath.ToSlash(filepath.Join("src/main/java", strings.Join(rootParts, "/")))
	}

	return rootPath, moduleID
}
