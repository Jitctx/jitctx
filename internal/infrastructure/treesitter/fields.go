package treesitter

import (
	"context"
	"fmt"
	"io/fs"

	sitter "github.com/smacker/go-tree-sitter"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
)

// ListJavaFields satisfies ListJavaFieldsPort. It parses the file and returns a
// JavaFileSummary populated with Path, Package, Imports, and
// Declarations[i].Fields only — Methods and other JavaDeclaration fields are
// left at their zero value so callers explicitly opt in to the field-only view.
//
// The same *Parser struct satisfies both ParseJavaFilePort and ListJavaFieldsPort
// (one collaborator, multiple ISP ports) without any shared mutable state.
func (p *Parser) ListJavaFields(ctx context.Context, fsys fs.FS, path string) (model.JavaFileSummary, error) {
	if err := ctx.Err(); err != nil {
		return model.JavaFileSummary{}, err
	}

	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return model.JavaFileSummary{}, fmt.Errorf("read file %s: %w", path, domerr.ErrParseFailure)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(JavaLanguage())
	tree, parseErr := parser.ParseCtx(ctx, nil, data)
	if parseErr != nil {
		return model.JavaFileSummary{}, fmt.Errorf("parse tree error for %s: %w", path, domerr.ErrParseFailure)
	}
	if tree == nil {
		return model.JavaFileSummary{}, fmt.Errorf("parse tree is nil for %s: %w", path, domerr.ErrParseFailure)
	}
	defer tree.Close()

	root := tree.RootNode()

	summary := model.JavaFileSummary{
		Path:      path,
		HasErrors: containsErrors(root),
	}

	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		switch child.Type() {
		case nodePackageDecl:
			summary.Package = extractPackage(child, data)
		case nodeImportDecl:
			imp := extractImport(child, data)
			if imp != "" {
				summary.Imports = append(summary.Imports, imp)
			}
		case nodeClassDecl:
			summary.Declarations = append(summary.Declarations, extractClassFields(child, data))
		}
	}

	if summary.HasErrors {
		return summary, fmt.Errorf("partial parse %s: %w", path, domerr.ErrPartialParse)
	}
	return summary, nil
}

// extractClassFields builds a thin JavaDeclaration that carries only the
// Name, NodeType, and Fields populated from direct class_body children.
// Methods, Annotations, Extends, and Implements are intentionally left zero
// — this function is for the field-only view consumed by ListJavaFields.
func extractClassFields(node *sitter.Node, src []byte) model.JavaDeclaration {
	decl := model.JavaDeclaration{NodeType: nodeClassDecl}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case nodeIdentifier:
			if decl.Name == "" {
				decl.Name = nodeText(child, src)
			}
		case nodeClassBody:
			decl.Fields = extractFields(child, src)
		}
	}
	return decl
}

// extractFields collects JavaField entries from the direct children of a
// class_body node. It does NOT recurse into nested classes or method bodies,
// which prevents inner-class fields from being double-counted (risk R-A).
//
// Multi-declarator fields (e.g. `private int a, b;`) are expanded into one
// JavaField per variable_declarator. Fields whose type identifier cannot be
// determined are silently skipped, mirroring the existing tolerance for
// malformed annotations in extractAnnotations.
func extractFields(bodyNode *sitter.Node, src []byte) []model.JavaField {
	var fields []model.JavaField
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		child := bodyNode.Child(i)
		if child.Type() != nodeFieldDecl {
			// Only direct field_declaration children — skip methods, nested
			// class declarations, constructors, etc.
			continue
		}
		fields = append(fields, extractFieldDeclaration(child, src)...)
	}
	return fields
}

// extractFieldDeclaration expands one field_declaration node into one or more
// JavaField values (one per variable_declarator).
func extractFieldDeclaration(node *sitter.Node, src []byte) []model.JavaField {
	// Determine the type of the field. The type node appears before the
	// variable_declarator(s). We accept the same set of type node kinds that
	// buildMethodSignature accepts.
	var fieldType string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case nodeTypeIdentifier, nodeIntegralType, nodeFloatingPointType, nodeBooleanType,
			nodeArrayType, nodeGenericType, nodeScopedTypeIdentifier, nodeVoidType:
			fieldType = nodeText(child, src)
		}
		if fieldType != "" {
			break
		}
	}

	if fieldType == "" {
		// Unparseable type — skip silently.
		return nil
	}

	// Collect all variable_declarator names (handles multi-declarator fields).
	var fields []model.JavaField
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != nodeVariableDeclarator {
			continue
		}
		name := extractVariableName(child, src)
		if name == "" {
			continue // malformed declarator — skip silently
		}
		fields = append(fields, model.JavaField{Name: name, Type: fieldType})
	}
	return fields
}

// extractVariableName returns the identifier name from a variable_declarator node.
// A variable_declarator node has the form: identifier (= initializer)?
func extractVariableName(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == nodeIdentifier {
			return nodeText(child, src)
		}
	}
	return ""
}
