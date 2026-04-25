package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestJavaIdentifierUtils(t *testing.T) {
	t.Parallel()

	utils := service.NewJavaIdentifierUtils()

	t.Run("FieldNameFromType_PascalCase", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "userRepository", utils.FieldNameFromType("UserRepository"))
	})

	t.Run("FieldNameFromType_AllCaps", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "uRLBuilder", utils.FieldNameFromType("URLBuilder"))
	})

	t.Run("FieldNameFromType_SingleChar", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "x", utils.FieldNameFromType("X"))
	})

	t.Run("FieldNameFromType_Empty", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "", utils.FieldNameFromType(""))
	})

	t.Run("PackageFromRelativePath_NestedPath", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "port.in", utils.PackageFromRelativePath("port/in/Foo.java"))
	})

	t.Run("PackageFromRelativePath_SingleSegment", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "", utils.PackageFromRelativePath("Foo.java"))
	})

	t.Run("PackageFromRelativePath_DeepPath", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "adapter.in.web", utils.PackageFromRelativePath("adapter/in/web/Bar.java"))
	})

	t.Run("FQN_NestedPackage", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "com.app.user.port.in.Foo", utils.FQN("com.app.user", "port/in/Foo.java", "Foo"))
	})

	t.Run("FQN_FlatPath", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "com.app.Foo", utils.FQN("com.app", "Foo.java", "Foo"))
	})
}
