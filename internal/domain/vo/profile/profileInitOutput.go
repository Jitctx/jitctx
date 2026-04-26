package profile

// ProfileInitOutput is the success result of profileinituc.Execute.
// Fields are presentation-friendly (the cobra layer renders them as
// "Initialised profile %q at %s (N files)").
type ProfileInitOutput struct {
	// Name of the bundled profile that was extracted.
	Name string

	// TargetDir is the absolute path of the freshly-created profile
	// directory.
	TargetDir string

	// FilesWritten is the count of files written under TargetDir,
	// including profile.yaml, README.md, and every template.
	FilesWritten int
}
