package model

import "time"

type Stack struct {
	Languages  []string
	Frameworks []string
}

type ProjectState struct {
	GeneratedAt time.Time
	Stack       Stack
	Modules     []Module
	Contexts    []Context
}

func NewProjectState(workDir string) *ProjectState {
	return &ProjectState{
		GeneratedAt: time.Now().UTC(),
	}
}

func (s *ProjectState) FindModule(id string) (*Module, bool) {
	for i := range s.Modules {
		if s.Modules[i].ID == id {
			return &s.Modules[i], true
		}
	}
	return nil, false
}
