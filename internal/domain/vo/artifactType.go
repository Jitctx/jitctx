package vo

import "fmt"

type ArtifactType string

const (
	ArtifactGuidelines   ArtifactType = "guidelines"
	ArtifactRequirements ArtifactType = "requirements"
	ArtifactScenarios    ArtifactType = "scenarios"
	ArtifactContracts    ArtifactType = "contracts"
)

func (a ArtifactType) Validate() error {
	switch a {
	case ArtifactGuidelines, ArtifactRequirements, ArtifactScenarios, ArtifactContracts:
		return nil
	}
	return fmt.Errorf("unknown artifact type: %q", string(a))
}
