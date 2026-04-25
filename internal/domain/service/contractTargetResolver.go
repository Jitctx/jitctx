package service

import (
	"errors"
	"path"
	"strings"
)

// ContractTargetResolver is a stateless pure function that derives the
// canonical contract name from a file path supplied by --for. The chosen
// rule (see §8 Q1, Option A) is: take the basename and strip the LAST
// extension. Examples:
//
//	src/main/java/com/app/UserServiceImpl.java   → "UserServiceImpl"
//	UserController.java                          → "UserController"
//	"User.java"                                  → "User"
//
// Rejected inputs (return non-nil error):
//   - empty string after trimming whitespace
//   - basename consisting only of an extension (".java")
//   - paths containing backslash separators (Windows-style not supported here)
//
// The function does NOT touch the filesystem and does NOT validate that the
// derived name corresponds to any known contract. Lookup is the use case's
// responsibility.
type ContractTargetResolver struct{}

// NewContractTargetResolver returns a stateless resolver.
func NewContractTargetResolver() ContractTargetResolver { return ContractTargetResolver{} }

var (
	errEmptyTargetFile      = errors.New("target file path must not be empty")
	errInvalidTargetFile    = errors.New("target file path produces empty contract name")
	errBackslashUnsupported = errors.New("target file path must use '/' separators")
)

// Resolve returns the contract name encoded in the basename of targetFile,
// or a non-nil error if the path is not interpretable. Uses path.Base and
// path.Ext (NOT filepath.*) so behaviour is stable across OSes — paths
// supplied to --for are expected to be POSIX-style relative paths from the
// project root (the convention used throughout EP-02 acceptance criteria).
func (ContractTargetResolver) Resolve(targetFile string) (string, error) {
	s := strings.TrimSpace(targetFile)
	if s == "" {
		return "", errEmptyTargetFile
	}
	if strings.ContainsRune(s, '\\') {
		return "", errBackslashUnsupported
	}
	base := path.Base(s)
	ext := path.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		return "", errInvalidTargetFile
	}
	return name, nil
}
