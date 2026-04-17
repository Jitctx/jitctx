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
