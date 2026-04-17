package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestModuleIDsSorted(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		modules []model.Module
		want    []string
	}{
		{
			name:    "empty-project-state",
			modules: nil,
			want:    []string{},
		},
		{
			name: "five-modules-in-reverse-order",
			modules: []model.Module{
				{ID: "user-management"},
				{ID: "payment"},
				{ID: "notification"},
				{ID: "inventory"},
				{ID: "auth"},
			},
			want: []string{"auth", "inventory", "notification", "payment", "user-management"},
		},
		{
			name:    "single-module",
			modules: []model.Module{{ID: "billing"}},
			want:    []string{"billing"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state := &model.ProjectState{Modules: tc.modules}
			got := service.ModuleIDsSorted(state)

			require.Len(t, got, len(tc.want))
			require.Equal(t, tc.want, got)
		})
	}
}
