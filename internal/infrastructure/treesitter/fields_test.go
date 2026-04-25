package treesitter_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/infrastructure/treesitter"
)

// TestListJavaFields_SimpleFields verifies that a class with two simple
// primitive/object fields produces two JavaField entries with the correct
// names and types.
func TestListJavaFields_SimpleFields(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.user.domain;

public class User {
    private int age;
    public String name;
}`
	fsys := fstest.MapFS{
		"User.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ListJavaFields(context.Background(), fsys, "User.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)

	fields := summary.Declarations[0].Fields
	require.Len(t, fields, 2)

	require.Equal(t, "age", fields[0].Name)
	require.Equal(t, "int", fields[0].Type)

	require.Equal(t, "name", fields[1].Name)
	require.Equal(t, "String", fields[1].Type)
}

// TestListJavaFields_MultiDeclaratorField verifies that `private int a, b;`
// is expanded into two separate JavaField entries, both with type "int".
func TestListJavaFields_MultiDeclaratorField(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.util;

public class Coords {
    private int x, y;
}`
	fsys := fstest.MapFS{
		"Coords.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ListJavaFields(context.Background(), fsys, "Coords.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)

	fields := summary.Declarations[0].Fields
	require.Len(t, fields, 2)

	require.Equal(t, "x", fields[0].Name)
	require.Equal(t, "int", fields[0].Type)

	require.Equal(t, "y", fields[1].Name)
	require.Equal(t, "int", fields[1].Type)
}

// TestListJavaFields_GenericFieldType verifies that a generic type such as
// List<String> is preserved verbatim in the JavaField.Type string.
func TestListJavaFields_GenericFieldType(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.order.domain;

import java.util.List;

public class Order {
    private List<String> items;
}`
	fsys := fstest.MapFS{
		"Order.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ListJavaFields(context.Background(), fsys, "Order.java")
	require.NoError(t, err)
	require.Len(t, summary.Declarations, 1)

	fields := summary.Declarations[0].Fields
	require.Len(t, fields, 1)
	require.Equal(t, "items", fields[0].Name)
	require.Equal(t, "List<String>", fields[0].Type)
}

// TestListJavaFields_InterfaceProducesNoFields verifies that an interface
// declaration produces a JavaFileSummary with no declarations (and therefore
// no fields), because ListJavaFields only walks class_declaration nodes.
func TestListJavaFields_InterfaceProducesNoFields(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.user.port.in;

public interface CreateUserUseCase {
    void execute(String name);
}`
	fsys := fstest.MapFS{
		"CreateUserUseCase.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ListJavaFields(context.Background(), fsys, "CreateUserUseCase.java")
	require.NoError(t, err)
	// ListJavaFields only processes class_declaration nodes; interfaces produce
	// no declarations in the field-only view.
	require.Empty(t, summary.Declarations)
}

// TestListJavaFields_NestedClassNoOverCount verifies that fields of an inner
// (nested) class are NOT included in the outer class's Fields slice. The outer
// class Fields slice must contain only its own directly declared fields.
func TestListJavaFields_NestedClassNoOverCount(t *testing.T) {
	t.Parallel()

	javaCode := `package com.app.report;

public class Report {
    private String title;

    public static class Summary {
        private int count;
        private double total;
    }
}`
	fsys := fstest.MapFS{
		"Report.java": &fstest.MapFile{Data: []byte(javaCode)},
	}

	p := treesitter.New()
	summary, err := p.ListJavaFields(context.Background(), fsys, "Report.java")
	require.NoError(t, err)
	// Only the top-level Report class_declaration appears in Declarations
	// (ListJavaFields walks only direct top-level class nodes).
	require.Len(t, summary.Declarations, 1)

	outerFields := summary.Declarations[0].Fields
	// The outer class has exactly one field: "title". Inner class fields
	// (count, total) must NOT appear here (risk R-A guard).
	require.Len(t, outerFields, 1)
	require.Equal(t, "title", outerFields[0].Name)
	require.Equal(t, "String", outerFields[0].Type)
}
