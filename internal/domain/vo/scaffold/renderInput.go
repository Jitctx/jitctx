package scaffold

// RenderInput is the fully-resolved view model handed to
// RenderProductionTemplatePort.Render. It is intentionally typed (not
// `map[string]any`) so the infra renderer cannot diverge silently from the
// view-model shape decided here.
//
// Field semantics:
//
//	ContractType    raw spec type string ("input-port", "service", ...);
//	                the renderer selects the .tmpl by this key.
//	Package         dot-form Java package for the generated file.
//	ClassName       PascalCase class/interface name (== contract Name).
//	Imports         distinct, alphabetically sorted FQN imports
//	                (computed by service.JavaImportResolver).
//	ClassAnnotations class-level annotations (e.g., "@Service", "@RestController")
//	                already prefixed with '@'; framework + spec-declared, deduped.
//	Implements      simple type name the class implements (services only); ""
//	                when not applicable.
//	Fields          declarations as written in the spec (e.g., "UUID id").
//	Methods         interface-method declarations (input-port / output-port)
//	                OR @Override stubs (service / jpa-adapter); rendered as-is
//	                plus a "throw new UnsupportedOperationException(...)" body
//	                for class types. The use case pre-builds these strings.
//	Endpoints       parsed Endpoint records (rest-adapter only).
//	Dependencies    constructor injection records (service / rest-adapter / jpa-adapter).
type RenderInput struct {
	ContractType     string
	Package          string
	ClassName        string
	Imports          []string
	ClassAnnotations []string
	Implements       string
	Fields           []string
	Methods          []RenderedMethod
	Endpoints        []RenderedEndpoint
	Dependencies     []ConstructorDep
}

// RenderedMethod is one interface declaration or @Override stub already
// formatted as the use case wants it to appear.
//
//	Signature: full Java signature WITHOUT trailing semicolon and WITHOUT
//	           braces (e.g., "UserResponse execute(CreateUserCommand cmd)").
//	Override:  true when the template should emit "@Override" above the method.
//	Body:      "" for interface methods (template emits a trailing ';').
//	           For class methods, the use case sets this to
//	           `throw new UnsupportedOperationException("Not yet implemented");`
type RenderedMethod struct {
	Signature string
	Override  bool
	Body      string
}

// RenderedEndpoint is one rest-adapter HTTP method binding.
//
//	Annotation: "@PostMapping(\"/users\")" — fully formed with quotes.
//	Method:     synthesised Java method name (e.g., "postUsers").
//	Body:       always `throw new UnsupportedOperationException(...)`.
type RenderedEndpoint struct {
	Annotation string
	Method     string
	Body       string
}

// ConstructorDep is one DI parameter on a service / rest-adapter / jpa-adapter.
//
//	Type:       simple Java type name (e.g., "UserRepository"). FQN imports
//	            are handled separately via Imports.
//	FieldName:  camelCase identifier (e.g., "userRepository").
type ConstructorDep struct {
	Type      string
	FieldName string
}
