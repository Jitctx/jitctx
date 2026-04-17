package model

import "github.com/jitctx/jitctx/internal/domain/vo"

type Context struct {
	ID            string
	Type          vo.ArtifactType
	AppliesTo     []string
	Module        string
	Tags          []string
	Path          string
	TokenEstimate int
}
