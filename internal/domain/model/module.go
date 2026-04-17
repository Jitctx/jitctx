package model

import (
	"errors"
	"strings"
)

type Module struct {
	ID           string
	Path         string
	Tags         []string
	Contracts    []Contract
	Dependencies []string
}

func NewModule(id, path string, tags []string) (*Module, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("module id must not be empty")
	}
	if strings.ContainsAny(id, " \t/") {
		return nil, errors.New("module id must be a kebab-case identifier")
	}
	return &Module{
		ID:   id,
		Path: path,
		Tags: append([]string(nil), tags...),
	}, nil
}

func (m *Module) HasTag(tag string) bool {
	for _, t := range m.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}
