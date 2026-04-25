package format_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/cli/format"
	contractsvo "github.com/jitctx/jitctx/internal/domain/vo/contracts"
)

func TestWriteContractsText_TargetWithMethods(t *testing.T) {
	t.Parallel()

	out := contractsvo.ExtractContractsOutput{
		Source: "spec",
		Target: contractsvo.ContractFragment{
			Name:    "UserServiceImpl",
			Type:    "service",
			Path:    "application/UserServiceImpl.java",
			Role:    "Service implementing CreateUserUseCase; depends on UserRepository",
			Methods: []string{"UserResponse execute(CreateUserCommand cmd)"},
		},
	}

	var buf bytes.Buffer
	err := format.WriteContractsText(&buf, out)
	require.NoError(t, err)

	got := buf.String()
	require.Contains(t, got, "# Target: UserServiceImpl (service)")
	require.Contains(t, got, "Source: spec")
	require.Contains(t, got, "Path: application/UserServiceImpl.java")
	require.Contains(t, got, "Role: Service implementing CreateUserUseCase; depends on UserRepository")
	require.Contains(t, got, "Methods:")
	require.Contains(t, got, "- UserResponse execute(CreateUserCommand cmd)")
}

func TestWriteContractsText_TargetAndDependencies(t *testing.T) {
	t.Parallel()

	out := contractsvo.ExtractContractsOutput{
		Source: "spec",
		Target: contractsvo.ContractFragment{
			Name: "UserServiceImpl",
			Type: "service",
			Path: "application/UserServiceImpl.java",
			Role: "Service implementing CreateUserUseCase; depends on UserRepository",
		},
		Related: []contractsvo.ContractFragment{
			{
				Name: "CreateUserUseCase",
				Type: "input-port",
				Path: "domain/port/in/CreateUserUseCase.java",
				Role: "Input port (use case interface)",
			},
			{
				Name: "UserRepository",
				Type: "output-port",
				Path: "domain/port/out/UserRepository.java",
				Role: "Output port (driven port)",
			},
		},
	}

	var buf bytes.Buffer
	err := format.WriteContractsText(&buf, out)
	require.NoError(t, err)

	got := buf.String()
	require.Contains(t, got, "## Dependencies (2)")
	require.Contains(t, got, "### CreateUserUseCase (input-port)")
	require.Contains(t, got, "### UserRepository (output-port)")

	// Assert ordering: CreateUserUseCase must appear before UserRepository
	idxCreate := bytes.Index(buf.Bytes(), []byte("### CreateUserUseCase"))
	idxRepo := bytes.Index(buf.Bytes(), []byte("### UserRepository"))
	require.Less(t, idxCreate, idxRepo)
}

func TestWriteContractsText_NoMethodsNoDeps(t *testing.T) {
	t.Parallel()

	out := contractsvo.ExtractContractsOutput{
		Source: "spec",
		Target: contractsvo.ContractFragment{
			Name:      "CreateUserUseCase",
			Type:      "input-port",
			Path:      "domain/port/in/CreateUserUseCase.java",
			Role:      "Input port (use case interface)",
			Methods:   nil,
			Fields:    nil,
			Endpoints: nil,
		},
		Related: nil,
	}

	var buf bytes.Buffer
	err := format.WriteContractsText(&buf, out)
	require.NoError(t, err)

	got := buf.String()
	require.Contains(t, got, "## Dependencies (0)")
	require.NotContains(t, got, "Methods:")
	require.NotContains(t, got, "Fields:")
	require.NotContains(t, got, "Endpoints:")
}

func TestWriteContractsJSON_RoundTrip(t *testing.T) {
	t.Parallel()

	out := contractsvo.ExtractContractsOutput{
		Source: "spec",
		Target: contractsvo.ContractFragment{
			Name:    "UserServiceImpl",
			Type:    "service",
			Path:    "application/UserServiceImpl.java",
			Role:    "Service implementing CreateUserUseCase; depends on UserRepository",
			Methods: []string{"UserResponse execute(CreateUserCommand cmd)"},
		},
		Related: []contractsvo.ContractFragment{
			{
				Name: "CreateUserUseCase",
				Type: "input-port",
				Path: "domain/port/in/CreateUserUseCase.java",
				Role: "Input port (use case interface)",
			},
		},
	}

	var buf bytes.Buffer
	err := format.WriteContractsJSON(&buf, out)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))

	require.Equal(t, "spec", decoded["source"])

	target, ok := decoded["target"].(map[string]any)
	require.True(t, ok, "target must be a JSON object")
	require.Equal(t, out.Target.Name, target["name"])

	related, ok := decoded["related"].([]any)
	require.True(t, ok, "related must be a JSON array")
	require.Len(t, related, 1)

	firstRelated, ok := related[0].(map[string]any)
	require.True(t, ok, "related[0] must be a JSON object")
	_, hasName := firstRelated["name"]
	require.True(t, hasName, "related[0] must contain a 'name' key")
}
