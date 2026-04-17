package treesitter

import (
	"context"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/domain/model"
)

// Parser implements ParseJavaFilePort.
type Parser struct{}

// New creates a new Parser.
func New() *Parser {
	return &Parser{}
}

// ParseJavaFile parses a single Java file and returns a JavaFileSummary.
// If the file has syntax errors, it returns the summary AND a wrapped ErrPartialParse.
func (p *Parser) ParseJavaFile(ctx context.Context, fsys fs.FS, path string) (model.JavaFileSummary, error) {
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
	hasErrors := containsErrors(root)

	summary := model.JavaFileSummary{
		Path:      path,
		HasErrors: hasErrors,
	}

	// Walk the top-level nodes to extract package, imports, and declarations.
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		switch child.Type() {
		case "package_declaration":
			summary.Package = extractPackage(child, data)
		case "import_declaration":
			imp := extractImport(child, data)
			if imp != "" {
				summary.Imports = append(summary.Imports, imp)
			}
		case "class_declaration":
			decl := extractClassDeclaration(child, data)
			summary.Declarations = append(summary.Declarations, decl)
		case "interface_declaration":
			decl := extractInterfaceDeclaration(child, data)
			summary.Declarations = append(summary.Declarations, decl)
		case "enum_declaration":
			decl := extractEnumDeclaration(child, data)
			summary.Declarations = append(summary.Declarations, decl)
		case "record_declaration":
			decl := extractRecordDeclaration(child, data)
			summary.Declarations = append(summary.Declarations, decl)
		}
	}

	if hasErrors {
		return summary, fmt.Errorf("partial parse %s: %w", path, domerr.ErrPartialParse)
	}
	return summary, nil
}

// containsErrors recursively checks if any node is an ERROR node.
func containsErrors(node *sitter.Node) bool {
	if node.IsError() || node.IsMissing() {
		return true
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		if containsErrors(node.Child(i)) {
			return true
		}
	}
	return false
}

// nodeText returns the source text for a node.
func nodeText(node *sitter.Node, src []byte) string {
	if node == nil {
		return ""
	}
	return string(src[node.StartByte():node.EndByte()])
}

// extractPackage extracts the package name from a package_declaration node.
func extractPackage(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		t := child.Type()
		if t == "scoped_identifier" || t == "identifier" {
			return nodeText(child, src)
		}
	}
	return ""
}

// extractImport extracts the fully-qualified import name from an import_declaration node.
func extractImport(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		t := child.Type()
		if t == "scoped_identifier" || t == "identifier" {
			return nodeText(child, src)
		}
	}
	return ""
}

// extractAnnotations collects annotation names from a modifiers node.
func extractAnnotations(node *sitter.Node, src []byte) []string {
	if node == nil {
		return nil
	}
	var annotations []string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "annotation", "marker_annotation", "normal_annotation":
			name := findChildByTypes(child, src, "identifier")
			if name != "" {
				annotations = append(annotations, name)
			}
		}
	}
	return annotations
}

// findChildByTypes finds the first child with any of the given types and returns its text.
func findChildByTypes(node *sitter.Node, src []byte, types ...string) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if slices.Contains(types, child.Type()) {
			return nodeText(child, src)
		}
	}
	return ""
}

// extractClassDeclaration extracts a class declaration node into a JavaDeclaration.
func extractClassDeclaration(node *sitter.Node, src []byte) model.JavaDeclaration {
	decl := model.JavaDeclaration{NodeType: "class_declaration"}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "modifiers":
			decl.Annotations = extractAnnotations(child, src)
		case "identifier":
			decl.Name = nodeText(child, src)
		case "superclass":
			// superclass has a type child
			for j := 0; j < int(child.ChildCount()); j++ {
				tc := child.Child(j)
				if tc.Type() == "type_identifier" || tc.Type() == "generic_type" {
					decl.Extends = append(decl.Extends, extractSimpleName(nodeText(tc, src)))
				}
			}
		case "super_interfaces":
			decl.Implements = extractTypeList(child, src)
		case "class_body":
			decl.Methods = extractMethods(child, src)
		}
	}
	return decl
}

// extractInterfaceDeclaration extracts an interface declaration node.
func extractInterfaceDeclaration(node *sitter.Node, src []byte) model.JavaDeclaration {
	decl := model.JavaDeclaration{NodeType: "interface_declaration"}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "modifiers":
			decl.Annotations = extractAnnotations(child, src)
		case "identifier":
			decl.Name = nodeText(child, src)
		case "extends_interfaces":
			decl.Extends = extractTypeList(child, src)
		case "interface_body":
			decl.Methods = extractMethods(child, src)
		}
	}
	return decl
}

// extractEnumDeclaration extracts an enum declaration node.
func extractEnumDeclaration(node *sitter.Node, src []byte) model.JavaDeclaration {
	decl := model.JavaDeclaration{NodeType: "enum_declaration"}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "modifiers":
			decl.Annotations = extractAnnotations(child, src)
		case "identifier":
			decl.Name = nodeText(child, src)
		case "super_interfaces":
			decl.Implements = extractTypeList(child, src)
		}
	}
	return decl
}

// extractRecordDeclaration extracts a record declaration node.
func extractRecordDeclaration(node *sitter.Node, src []byte) model.JavaDeclaration {
	decl := model.JavaDeclaration{NodeType: "record_declaration"}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "modifiers":
			decl.Annotations = extractAnnotations(child, src)
		case "identifier":
			decl.Name = nodeText(child, src)
		case "super_interfaces":
			decl.Implements = extractTypeList(child, src)
		}
	}
	return decl
}

// extractTypeList extracts type names from a super_interfaces or extends_interfaces node.
func extractTypeList(node *sitter.Node, src []byte) []string {
	var types []string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_list" || child.Type() == "interface_type_list" {
			for j := 0; j < int(child.ChildCount()); j++ {
				tc := child.Child(j)
				switch tc.Type() {
				case "type_identifier":
					types = append(types, nodeText(tc, src))
				case "generic_type":
					// Strip generics for the implements check: List<User> → List
					types = append(types, extractSimpleName(nodeText(tc, src)))
				}
			}
		}
	}
	return types
}

// extractSimpleName strips generic type parameters: "List<User>" → "List".
func extractSimpleName(typeName string) string {
	if before, _, found := strings.Cut(typeName, "<"); found {
		return strings.TrimSpace(before)
	}
	return strings.TrimSpace(typeName)
}

// extractMethods extracts method declarations from a class_body or interface_body node.
func extractMethods(bodyNode *sitter.Node, src []byte) []model.JavaMethod {
	var methods []model.JavaMethod
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		child := bodyNode.Child(i)
		if child.Type() == "method_declaration" {
			sig := buildMethodSignature(child, src)
			if sig != "" {
				methods = append(methods, model.JavaMethod{Signature: sig})
			}
		}
	}
	return methods
}

// buildMethodSignature reconstructs "ReturnType name(params)" from a method_declaration node.
func buildMethodSignature(node *sitter.Node, src []byte) string {
	var returnType, name, params string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "void_type":
			returnType = "void"
		case "type_identifier", "integral_type", "floating_point_type", "boolean_type",
			"array_type", "generic_type":
			returnType = nodeText(child, src)
		case "identifier":
			if name == "" { // first identifier is the method name
				name = nodeText(child, src)
			}
		case "formal_parameters":
			params = extractFormalParams(child, src)
		}
	}

	if name == "" || returnType == "" {
		return ""
	}
	return fmt.Sprintf("%s %s(%s)", returnType, name, params)
}

// extractFormalParams extracts the parameter list text from formal_parameters, excluding outer parens.
func extractFormalParams(node *sitter.Node, src []byte) string {
	var parts []string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "formal_parameter", "spread_parameter":
			parts = append(parts, extractSingleParam(child, src))
		}
	}
	return strings.Join(parts, ", ")
}

// extractSingleParam builds "Type name" from a formal_parameter node.
func extractSingleParam(node *sitter.Node, src []byte) string {
	var paramType, paramName string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "type_identifier", "integral_type", "floating_point_type", "boolean_type",
			"array_type", "generic_type", "void_type":
			paramType = nodeText(child, src)
		case "variable_declarator_id":
			// variable_declarator_id has an identifier child
			for j := 0; j < int(child.ChildCount()); j++ {
				gc := child.Child(j)
				if gc.Type() == "identifier" {
					paramName = nodeText(gc, src)
				}
			}
		case "identifier":
			if paramType != "" && paramName == "" {
				paramName = nodeText(child, src)
			}
		case "modifiers":
			// skip final/annotations on parameters
		}
	}
	if paramType == "" {
		return strings.TrimSpace(nodeText(node, src))
	}
	if paramName == "" {
		return paramType
	}
	return paramType + " " + paramName
}
