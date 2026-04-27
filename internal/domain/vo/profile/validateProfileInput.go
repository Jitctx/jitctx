package profile

// ValidateProfileInput is the input VO for profilevalidateuc.UseCase.
//
// Path must point to a profile DIRECTORY (the same shape consumed by
// profile.LoadProfileBundlePort with Dir set). Path is resolved via
// filepath.Abs at the cobra layer; the use case treats it verbatim.
type ValidateProfileInput struct {
	Path string
}
