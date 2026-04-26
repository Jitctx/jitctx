package profile

// LoadProfileBundleInput is the input VO for profile.LoadProfileBundlePort.
// One of Dir or BundledName must be non-empty; the loader uses Dir when
// non-empty and falls back to BundledName otherwise.
type LoadProfileBundleInput struct {
	// Dir is an absolute or project-relative path to a profile directory.
	// When non-empty, the loader treats this as a filesystem load.
	Dir string

	// BundledName, when Dir is empty, names a profile to load from the
	// binary embed. Must match a directory name under fsprofile/bundled/.
	BundledName string
}
