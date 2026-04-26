package profile

import (
	"context"

	"github.com/jitctx/jitctx/internal/domain/model"
	profilevo "github.com/jitctx/jitctx/internal/domain/vo/profile"
)

// ResolveProfilePort selects which profile to load for a project per
// EP04RF-012's resolution order:
//
//  1. If Input.Name != "", try the user directory
//     <ProfilesDir>/<Name>/ first; if absent, try the bundled
//     profile <Name>; if both miss, return ErrProfileNotFound.
//  2. If Input.Name == "", iterate user-dir profiles in
//     <ProfilesDir>/ in alphabetical order; return the first one
//     that loads cleanly. If none loads, iterate bundled profiles
//     via ListBundledProfilesPort in alphabetical order; return
//     the first one that loads. If neither produces a hit, return
//     ErrNoProfileMatch.
//
// The returned bundle's Profile.Source MUST be ProfileSourceCustom
// when the user-dir branch produced it and ProfileSourceBundled
// when the embed branch produced it. The scan log line keys off
// this field.
//
// Implementations MUST NOT consult the legacy single-file YAML
// profiles handled by DetectProfilePort — that path is the audit /
// refactor world and stays separate during EP04US-006.
type ResolveProfilePort interface {
	Resolve(ctx context.Context, input profilevo.ResolveProfileInput) (*model.ProfileBundle, error)
}
