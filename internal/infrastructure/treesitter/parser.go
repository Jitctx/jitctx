package treesitter

import (
	"context"
	"fmt"
	"io/fs"
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
		case nodePackageDecl:
			summary.Package = extractPackage(child, data)
		case nodeImportDecl:
			imp := extractImport(child, data)
			if imp != "" {
				summary.Imports = append(summary.Imports, imp)
			}
		case nodeClassDecl:
			decl := extractClassDeclaration(child, data)
			summary.Declarations = append(summary.Declarations, decl)
		case nodeInterfaceDecl:
			decl := extractInterfaceDeclaration(child, data)
			summary.Declarations = append(summary.Declarations, decl)
		case nodeEnumDecl:
			decl := extractEnumDeclaration(child, data)
			summary.Declarations = append(summary.Declarations, decl)
		case nodeRecordDecl:
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
		if t == nodeScopedIdentifier || t == nodeIdentifier {
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
		if t == nodeScopedIdentifier || t == nodeIdentifier {
			return nodeText(child, src)
		}
	}
	return ""
}

// extractAnnotations collects annotation names from a modifiers node.
// It returns two parallel slices of equal length: simple names and fully-qualified names.
// When the annotation was written as a simple name (e.g. @Entity), both slices carry
// the same value. When written as a scoped name (e.g. @jakarta.persistence.Entity),
// the simple slice carries "Entity" and the qualified slice carries
// "jakarta.persistence.Entity". Malformed annotations are silently skipped to preserve
// the invariant len(simple) == len(qualified).
func extractAnnotations(node *sitter.Node, src []byte) (simple, qualified []string) {
	if node == nil {
		return nil, nil
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case nodeAnnotation, nodeMarkerAnnotation, nodeNormalAnnotation:
			raw := findAnnotationNameChild(child, src)
			if raw == "" {
				continue // malformed — skip to keep invariant len(simple) == len(qualified)
			}
			simple = append(simple, terminalSegment(raw))
			qualified = append(qualified, raw)
		}
	}
	return
}

// findAnnotationNameChild returns the source text of the first
// type_identifier | identifier | scoped_identifier child of an annotation node,
// or "" when none is found.
func findAnnotationNameChild(ann *sitter.Node, src []byte) string {
	for j := 0; j < int(ann.ChildCount()); j++ {
		c := ann.Child(j)
		switch c.Type() {
		case nodeTypeIdentifier, nodeIdentifier, nodeScopedIdentifier:
			return nodeText(c, src)
		}
	}
	return ""
}

// terminalSegment returns the trailing ".xxx" segment of a dotted name,
// or the name itself when no dot is present.
func terminalSegment(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// extractClassDeclaration extracts a class declaration node into a JavaDeclaration.
func extractClassDeclaration(node *sitter.Node, src []byte) model.JavaDeclaration {
	decl := model.JavaDeclaration{NodeType: nodeClassDecl}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case nodeModifiers:
			decl.Annotations, decl.QualifiedAnnotations = extractAnnotations(child, src)
		case nodeIdentifier:
			decl.Name = nodeText(child, src)
		case nodeSuperclass:
			// superclass has a type child
			for j := 0; j < int(child.ChildCount()); j++ {
				tc := child.Child(j)
				if tc.Type() == nodeTypeIdentifier || tc.Type() == nodeGenericType {
					decl.Extends = append(decl.Extends, extractSimpleName(nodeText(tc, src)))
				}
			}
		case nodeSuperInterfaces:
			decl.Implements = extractTypeList(child, src)
		case nodeClassBody:
			decl.Methods = extractMethods(child, src)
			decl.Fields = extractFields(child, src)
		}
	}
	return decl
}

// extractInterfaceDeclaration extracts an interface declaration node.
func extractInterfaceDeclaration(node *sitter.Node, src []byte) model.JavaDeclaration {
	decl := model.JavaDeclaration{NodeType: nodeInterfaceDecl}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case nodeModifiers:
			decl.Annotations, decl.QualifiedAnnotations = extractAnnotations(child, src)
		case nodeIdentifier:
			decl.Name = nodeText(child, src)
		case nodeExtendsInterfaces:
			decl.Extends = extractTypeList(child, src)
		case nodeInterfaceBody:
			decl.Methods = extractMethods(child, src)
		}
	}
	return decl
}

// extractEnumDeclaration extracts an enum declaration node.
func extractEnumDeclaration(node *sitter.Node, src []byte) model.JavaDeclaration {
	decl := model.JavaDeclaration{NodeType: nodeEnumDecl}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case nodeModifiers:
			decl.Annotations, decl.QualifiedAnnotations = extractAnnotations(child, src)
		case nodeIdentifier:
			decl.Name = nodeText(child, src)
		case nodeSuperInterfaces:
			decl.Implements = extractTypeList(child, src)
		}
	}
	return decl
}

// extractRecordDeclaration extracts a record declaration node.
func extractRecordDeclaration(node *sitter.Node, src []byte) model.JavaDeclaration {
	decl := model.JavaDeclaration{NodeType: nodeRecordDecl}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case nodeModifiers:
			decl.Annotations, decl.QualifiedAnnotations = extractAnnotations(child, src)
		case nodeIdentifier:
			decl.Name = nodeText(child, src)
		case nodeSuperInterfaces:
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
		if child.Type() == nodeTypeList || child.Type() == nodeInterfaceTypeList {
			for j := 0; j < int(child.ChildCount()); j++ {
				tc := child.Child(j)
				switch tc.Type() {
				case nodeTypeIdentifier:
					types = append(types, nodeText(tc, src))
				case nodeGenericType:
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
		if child.Type() == nodeMethodDecl {
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
		case nodeVoidType:
			returnType = "void"
		case nodeTypeIdentifier, nodeIntegralType, nodeFloatingPointType, nodeBooleanType,
			nodeArrayType, nodeGenericType, nodeScopedTypeIdentifier:
			returnType = nodeText(child, src)
		case nodeIdentifier:
			if name == "" { // first identifier is the method name
				name = nodeText(child, src)
			}
		case nodeFormalParameters:
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
		case nodeFormalParameter, nodeSpreadParameter:
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
		case nodeTypeIdentifier, nodeIntegralType, nodeFloatingPointType, nodeBooleanType,
			nodeArrayType, nodeGenericType, nodeVoidType, nodeScopedTypeIdentifier:
			paramType = nodeText(child, src)
		case nodeVariableDeclaratorID:
			// variable_declarator_id has an identifier child
			for j := 0; j < int(child.ChildCount()); j++ {
				gc := child.Child(j)
				if gc.Type() == nodeIdentifier {
					paramName = nodeText(gc, src)
				}
			}
		case nodeIdentifier:
			if paramType != "" && paramName == "" {
				paramName = nodeText(child, src)
			}
		case nodeModifiers:
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
