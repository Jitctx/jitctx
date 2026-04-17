package profile

import "context"

type ListProfilesPort interface {
	List(ctx context.Context) ([]string, error)
}
