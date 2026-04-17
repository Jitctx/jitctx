package format

import (
	"io"

	"gopkg.in/yaml.v3"

	queryvo "github.com/jitctx/jitctx/internal/domain/vo/query"
)

// writeQueryYAML serialises a QueryContextOutput as a YAML document on w.
// The DTO translation (Body -> content, etc.) is handled locally so the
// domain VO is not polluted with yaml struct tags (CLAUDE.md rule).
func writeQueryYAML(w io.Writer, out queryvo.QueryContextOutput) error {
	doc := newQueryYAMLDoc(out)
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		_ = enc.Close()
		return err
	}
	return enc.Close()
}

type queryYAMLDoc struct {
	Metadata queryYAMLMetadata  `yaml:"metadata"`
	Contexts []queryYAMLContext `yaml:"contexts"`
}

type queryYAMLMetadata struct {
	Module       string                     `yaml:"module"`
	ContextCount int                        `yaml:"context_count"`
	TokenTotal   int                        `yaml:"token_total"`
	TrimmedCount int                        `yaml:"trimmed_count"`
	Contracts    []queryYAMLContractSummary `yaml:"contracts,omitempty"`
}

type queryYAMLContractSummary struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Methods []string `yaml:"methods,omitempty"`
}

type queryYAMLContext struct {
	Path          string   `yaml:"path"`
	Type          string   `yaml:"type"`
	Tags          []string `yaml:"tags"`
	TokenEstimate int      `yaml:"token_estimate"`
	Content       string   `yaml:"content"`
}

func newQueryYAMLDoc(out queryvo.QueryContextOutput) queryYAMLDoc {
	contexts := make([]queryYAMLContext, 0, len(out.Loaded))
	for _, c := range out.Loaded {
		tags := c.Tags
		if tags == nil {
			tags = []string{}
		}
		contexts = append(contexts, queryYAMLContext{
			Path:          c.Path,
			Type:          string(c.Type),
			Tags:          tags,
			TokenEstimate: c.TokenEstimate,
			Content:       c.Body,
		})
	}

	var contracts []queryYAMLContractSummary
	if len(out.Module.Contracts) > 0 {
		contracts = make([]queryYAMLContractSummary, 0, len(out.Module.Contracts))
		for _, cc := range out.Module.Contracts {
			methods := cc.Methods
			if methods == nil {
				methods = []string{}
			}
			contracts = append(contracts, queryYAMLContractSummary{
				Name:    cc.Name,
				Type:    cc.Type,
				Methods: methods,
			})
		}
	}

	return queryYAMLDoc{
		Metadata: queryYAMLMetadata{
			Module:       out.Module.ID,
			ContextCount: len(out.Loaded),
			TokenTotal:   out.TotalTokens,
			TrimmedCount: len(out.Trimmed),
			Contracts:    contracts,
		},
		Contexts: contexts,
	}
}
