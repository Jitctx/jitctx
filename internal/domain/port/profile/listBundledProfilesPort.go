package profile

import "context"

// ListBundledProfilesPort returns the names of every profile embedded in
// the binary, sorted alphabetically. Used by US-006 (init) and by error
// messages that need to list available bundled names. Introduced now to
// keep the bundled-profile interface coherent in a single US.
type ListBundledProfilesPort interface {
	ListBundled(ctx context.Context) ([]string, error)
}
