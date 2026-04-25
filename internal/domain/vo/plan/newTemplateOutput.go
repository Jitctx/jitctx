package plan

// NewTemplateOutput reports the final on-disk path that was successfully
// written. The path is the one returned by the writer port (which is
// responsible for atomic rename).
type NewTemplateOutput struct {
	Path string // absolute or workdir-relative path of the created spec file
}
