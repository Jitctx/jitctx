package treesitter

// Tree-sitter Java grammar node type names consumed by parser.go.
// Centralized here so a future migration to a .scm-based query layer
// has a single seam to replace.
const (
	nodePackageDecl   = "package_declaration"
	nodeImportDecl    = "import_declaration"
	nodeClassDecl     = "class_declaration"
	nodeInterfaceDecl = "interface_declaration"
	nodeEnumDecl      = "enum_declaration"
	nodeRecordDecl    = "record_declaration"

	nodeModifiers        = "modifiers"
	nodeAnnotation       = "annotation"
	nodeMarkerAnnotation = "marker_annotation"
	nodeNormalAnnotation = "normal_annotation"

	nodeIdentifier           = "identifier"
	nodeTypeIdentifier       = "type_identifier"
	nodeScopedIdentifier     = "scoped_identifier"
	nodeScopedTypeIdentifier = "scoped_type_identifier"

	nodeSuperclass        = "superclass"
	nodeSuperInterfaces   = "super_interfaces"
	nodeExtendsInterfaces = "extends_interfaces"
	nodeTypeList          = "type_list"
	nodeInterfaceTypeList = "interface_type_list"
	nodeClassBody         = "class_body"
	nodeInterfaceBody     = "interface_body"

	nodeMethodDecl           = "method_declaration"
	nodeFormalParameters     = "formal_parameters"
	nodeFormalParameter      = "formal_parameter"
	nodeSpreadParameter      = "spread_parameter"
	nodeVariableDeclaratorID = "variable_declarator_id"

	nodeVoidType          = "void_type"
	nodeIntegralType      = "integral_type"
	nodeFloatingPointType = "floating_point_type"
	nodeBooleanType       = "boolean_type"
	nodeArrayType         = "array_type"
	nodeGenericType       = "generic_type"
)
