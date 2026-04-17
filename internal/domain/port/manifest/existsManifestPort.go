package manifest

import "context"

type ExistsManifestPort interface {
	Exists(ctx context.Context) (bool, error)
}
