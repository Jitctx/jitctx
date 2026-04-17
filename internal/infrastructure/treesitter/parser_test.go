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
