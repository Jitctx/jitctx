package treesitter_test

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	domerr "github.com/jitctx/jitctx/internal/domain/errors"
	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"
)

func TestParser_ClassWithAnnotation(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.user.domain;
import jakarta.persistence.Entity;

@Entity
public class User {
    private Long id;
}`
	fsys := fstest.MapFS{
		"User.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "User.java")
	require.NoError(t, err)
	require.Equal(t, "com.app.user.domain", summary.Package)
	require.Len(t, summary.Declarations, 1)
	require.Equal(t, "class_declaration", summary.Declarations[0].NodeType)
	require.Equal(t, "User", summary.Declarations[0].Name)
	require.Contains(t, summary.Declarations[0].Annotations, "Entity")
}

func TestParser_InterfaceWithMethods(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.user.port.in;

public interface CreateUserUseCase {
    User execute(String name, String email);
}`
	fsys := fstest.MapFS{
		"CreateUserUseCase.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "CreateUserUseCase.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)
	decl := summary.Declarations[0]
	require.Equal(t, "interface_declaration", decl.NodeType)
	require.Equal(t, "CreateUserUseCase", decl.Name)
	require.Len(t, decl.Methods, 1)
	require.Equal(t, "User execute(String name, String email)", decl.Methods[0].Signature)
}

func TestParser_Imports(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.user.service;
import com.app.notification.port.in.SendNotificationUseCase;
import java.util.List;

public class UserServiceImpl implements CreateUserUseCase {
}`
	fsys := fstest.MapFS{
		"UserServiceImpl.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "UserServiceImpl.java")
	require.NoError(t, err)
	require.Contains(t, summary.Imports, "com.app.notification.port.in.SendNotificationUseCase")
	require.Contains(t, summary.Imports, "java.util.List")
}

func TestParser_PartialParse(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.broken;

public class Broken {
    public void doSomething() {
        // unclosed brace`
	fsys := fstest.MapFS{
		"Broken.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "Broken.java")
	require.True(t, errors.Is(err, domerr.ErrPartialParse))
	require.True(t, summary.HasErrors)
}

func TestParser_ImplementsInterface(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.user.service;

public class UserServiceImpl implements CreateUserUseCase {
    public User execute(String name, String email) {
        return null;
    }
}`
	fsys := fstest.MapFS{
		"UserServiceImpl.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "UserServiceImpl.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)
	require.Contains(t, summary.Declarations[0].Implements, "CreateUserUseCase")
}

func TestParser_MultipleAnnotationsWithArguments(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.user.domain;

import jakarta.persistence.Entity;
import jakarta.persistence.Table;

@Entity
@Table(name = "users")
public class User {}`
	fsys := fstest.MapFS{
		"User.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "User.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)
	decl := summary.Declarations[0]
	require.Equal(t, []string{"Entity", "Table"}, decl.Annotations)
	require.Equal(t, []string{"Entity", "Table"}, decl.QualifiedAnnotations)
}

func TestParser_QualifiedAnnotation(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.user.domain;

@jakarta.persistence.Entity
public class User {}`
	fsys := fstest.MapFS{
		"User.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "User.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)
	decl := summary.Declarations[0]
	require.Equal(t, []string{"Entity"}, decl.Annotations)
	require.Equal(t, []string{"jakarta.persistence.Entity"}, decl.QualifiedAnnotations)
}

func TestParser_MethodReturnGeneric(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.user.port.out;

import java.util.Optional;

public interface UserRepository {
    Optional<User> findByEmail(String email);
}`
	fsys := fstest.MapFS{
		"UserRepository.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "UserRepository.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)
	decl := summary.Declarations[0]
	require.Len(t, decl.Methods, 1)
	require.Equal(t, "Optional<User> findByEmail(String email)", decl.Methods[0].Signature)
}

func TestParser_MethodParamGeneric(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.user.port.in;

public interface CreateBatch {
    void apply(java.util.List<String> names);
}`
	fsys := fstest.MapFS{
		"CreateBatch.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "CreateBatch.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)
	decl := summary.Declarations[0]
	require.Len(t, decl.Methods, 1)
	require.Equal(t, "void apply(java.util.List<String> names)", decl.Methods[0].Signature)
}

func TestParser_ImportStatic(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.util;

import static java.util.Collections.emptyList;
import com.app.notification.port.in.SendNotificationUseCase;

public class Helper {}`
	fsys := fstest.MapFS{
		"Helper.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "Helper.java")
	require.NoError(t, err)
	require.Contains(t, summary.Imports, "com.app.notification.port.in.SendNotificationUseCase")
	require.Contains(t, summary.Imports, "java.util.Collections.emptyList")
}

func TestParser_ImportWildcard(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.util;

import java.util.*;

public class Helper {}`
	fsys := fstest.MapFS{
		"Helper.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "Helper.java")
	require.NoError(t, err)
	require.Contains(t, summary.Imports, "java.util")
}

// TestParser_MethodAnnotationsNameLine verifies that extractMethods populates
// JavaMethod.Name, JavaMethod.Annotations, and JavaMethod.Line from the AST.
// R-001 mitigation: locks the producer contract for methods with trigger annotations.
func TestParser_MethodAnnotationsNameLine(t *testing.T) {
	t.Parallel()

	javaCode := `package com.acme;

import org.junit.jupiter.api.Test;

public class UserServiceTest {

    @Test
    public void testFindUser() {
    }
}`
	fsys := fstest.MapFS{
		"UserServiceTest.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "UserServiceTest.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)
	decl := summary.Declarations[0]
	require.Len(t, decl.Methods, 1)
	method := decl.Methods[0]
	require.Equal(t, "testFindUser", method.Name)
	require.Equal(t, []string{"Test"}, method.Annotations)
	// Tree-sitter method_declaration includes the modifiers child (the @Test line),
	// so StartPoint().Row is the annotation line (0-based row 6 → 1-based line 7).
	require.Equal(t, 7, method.Line)
}

func TestParser_PartialParseSurfacesErrorAndKeepsValidDeclarations(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.mixed;

public class Good {
    public void doGood() { }
}

public class Bad {
    public void doBad(`
	fsys := fstest.MapFS{
		"Mixed.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "Mixed.java")
	require.True(t, errors.Is(err, domerr.ErrPartialParse))
	require.True(t, summary.HasErrors)
	require.NotEmpty(t, summary.Declarations)

	// The valid "Good" class must be present among the declarations.
	var found bool
	for _, d := range summary.Declarations {
		if d.NodeType == "class_declaration" && d.Name == "Good" {
			found = true
			break
		}
	}
	require.True(t, found, "expected 'Good' class declaration to be present in partial parse output")
}

// TestParser_ClassWithExtendWithArg_PopulatesAnnotationArgs verifies that
// JavaDeclaration.AnnotationArgs is populated for annotations that carry a
// positional argument.  Locks PC01RF-007 Q2: string literals retain their
// surrounding quotes and class literals retain the ".class" suffix.
func TestParser_ClassWithExtendWithArg_PopulatesAnnotationArgs(t *testing.T) {
	t.Parallel()

	javaCode := `package com.acme;

@ExtendWith(MockitoExtension.class)
@DisplayName("User service tests")
public class UserServiceTest {
}`
	fsys := fstest.MapFS{
		"UserServiceTest.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "UserServiceTest.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)
	decl := summary.Declarations[0]

	require.Equal(t, "MockitoExtension.class", decl.AnnotationArgs["ExtendWith"],
		"class literal arg must include the .class suffix verbatim")
	require.Equal(t, `"User service tests"`, decl.AnnotationArgs["DisplayName"],
		"string literal arg must preserve surrounding double-quotes")
}

// TestParser_ClassWithMarkerAnnotation_AnnotationArgEmpty verifies that a
// marker annotation (no argument list) produces an empty string entry in
// AnnotationArgs while still appearing in Annotations.
func TestParser_ClassWithMarkerAnnotation_AnnotationArgEmpty(t *testing.T) {
	t.Parallel()

	javaCode := `package com.acme;

@Deprecated
public class Old {
}`
	fsys := fstest.MapFS{
		"Old.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "Old.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)
	decl := summary.Declarations[0]

	require.Contains(t, decl.Annotations, "Deprecated",
		"marker annotation must appear in Annotations slice")
	require.Equal(t, "", decl.AnnotationArgs["Deprecated"],
		"marker annotation must produce an empty-string arg entry")
}

// TestParser_ClassWithoutAnnotations_AnnotationArgsNilOrEmpty verifies that a
// class with no annotations results in a nil or empty AnnotationArgs map.
func TestParser_ClassWithoutAnnotations_AnnotationArgsNilOrEmpty(t *testing.T) {
	t.Parallel()

	javaCode := `package com.acme;

public class Plain {
}`
	fsys := fstest.MapFS{
		"Plain.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ParseJavaFile(context.Background(), fsys, "Plain.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)
	decl := summary.Declarations[0]

	require.Empty(t, decl.AnnotationArgs,
		"class with no annotations must have nil or empty AnnotationArgs")
}
