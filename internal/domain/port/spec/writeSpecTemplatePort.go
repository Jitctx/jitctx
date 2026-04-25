package spec

import "context"

// WriteSpecTemplatePort writes a fully-rendered spec template to disk at
// the given path. It MUST:
//   - create parent directories as needed (mkdir -p)
//   - fail with domerr.ErrSpecFileExists when path already exists
//   - perform a temp-file write + atomic rename so a panic mid-write
//     never leaves a partial spec on disk
//
// The returned string is the canonicalised path actually written
// (typically the input path; included for output-VO symmetry).
type WriteSpecTemplatePort interface {
	Write(ctx context.Context, path string, content []byte) (string, error)
}
