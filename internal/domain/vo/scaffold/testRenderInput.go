package scaffold

// TestRenderInput is the view model handed to RenderTestTemplatePort.Render.
// It is intentionally distinct from RenderInput because (a) test templates
// consume a strict SUBSET (no Endpoints, no Fields, no Methods-with-bodies)
// and (b) tests need a class-under-test simple-name for @InjectMocks even
// for entity/aggregate templates. Keeping a separate VO prevents silent
// drift if RenderInput grows production-only fields.
//
//	ContractType   raw spec type string ("service", "rest-adapter",
//	               "entity", "aggregate-root"); the renderer selects the
//	               test .tmpl by this key.
//	Package        dot-form Java package; identical to the production
//	               class's package (test mirrors prod package).
//	ClassName      simple name of the class under test (e.g.,
//	               "UserServiceImpl"). The test class is "<ClassName>Test".
//	Imports        distinct, alphabetically sorted FQN imports. Always
//	               contains JUnit + (when relevant) Mockito FQNs.
//	Mocks          one entry per dependency the test must wire as @Mock
//	               (service: DependsOn; rest-adapter: dedup(Uses+DependsOn)).
//	               Empty for entity / aggregate-root.
//	TestMethods    one entry per public method on the class under test:
//	               for service / rest-adapter the use case maps each
//	               ParsedMethod.Name → "<name>_shouldDoSomething";
//	               for entity / aggregate-root the use case emits exactly
//	               one entry "placeholder_shouldDoSomething" (§8 Q3).
type TestRenderInput struct {
	ContractType string
	Package      string
	ClassName    string
	Imports      []string
	Mocks        []TestMockField
	TestMethods  []TestMethod
}

// TestMockField is one @Mock field: simple type name + camelCase field name.
//
//	Type:      simple Java type (e.g., "UserRepository").
//	FieldName: camelCase identifier (e.g., "userRepository").
type TestMockField struct {
	Type      string
	FieldName string
}

// TestMethod is one @Test method to emit. Body is always
//
//	"// TODO: implement test"
//
// for this story; kept as a field so future stories can inject richer
// scaffolding without changing the template signature.
type TestMethod struct {
	Name string
	Body string
}
