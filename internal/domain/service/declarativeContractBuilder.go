package service

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/port/profile"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// ClassifyAndBuildContracts iterates a JavaFileSummary's declarations,
// classifies each one against the profile's declarative types via the
// supplied classifier port, and returns the resulting []model.Contract
// with Types populated.
//
// Behaviour contract:
//   - For each declaration, builds a profilevo.ClassificationInput from
//     decl.NodeType, decl.Name, decl.Annotations, decl.Implements, and
//     summary.Path. Calls classifier.ClassifyDeclarative(ctx, in,
//     typesDecl).
//   - The returned []string is assigned verbatim to Contract.Types.
//   - Empty result ([]string{}) is preserved — the contract is still
//     emitted with Types: []. RF-005 Scenario 2 ("manifest entry has
//     types=[]") demands this.
//   - When typesDecl is empty, returns (nil, nil). The caller is
//     responsible for the legacy classifier fallback path (this is
//     the seam scanuc/BuildModules uses to keep EP-03 manifests
//     producible until US-009 ratifies the cutover).
//   - ctx.Err() is honoured at the top of the function and after each
//     classifier call.
//   - Methods are projected from decl.Methods directly (mirrors
//     BuildModules' existing behaviour).
//
// The function is pure modulo the classifier port; given a fake port
// returning a deterministic []string per input, the output is fully
// determined by (summary, typesDecl).
func ClassifyAndBuildContracts(
	ctx context.Context,
	classifier profile.ClassifyDeclarativePort,
	summary model.JavaFileSummary,
	typesDecl []model.ProfileTypeDeclaration,
) ([]model.Contract, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(typesDecl) == 0 {
		return nil, nil
	}
	out := make([]model.Contract, 0, len(summary.Declarations))
	for _, decl := range summary.Declarations {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		in := profilevo.ClassificationInput{
			Kind:        decl.NodeType,
			Name:        decl.Name,
			Annotations: decl.Annotations,
			Implements:  decl.Implements,
			Path:        summary.Path,
		}
		types, err := classifier.ClassifyDeclarative(ctx, in, typesDecl)
		if err != nil {
			return nil, err
		}
		if types == nil {
			// Normalise to non-nil empty slice so YAML emits `types: []`.
			types = []string{}
		}
		methods := make([]model.Method, 0, len(decl.Methods))
		for _, m := range decl.Methods {
			methods = append(methods, model.Method{Signature: m.Signature})
		}
		out = append(out, model.Contract{
			Name:    decl.Name,
			Types:   types,
			Path:    summary.Path,
			Methods: methods,
		})
	}
	return out, nil
}
