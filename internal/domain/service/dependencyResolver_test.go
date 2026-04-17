package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jitctx/jitctx/internal/domain/model"
	"github.com/jitctx/jitctx/internal/domain/service"
)

func TestResolveDependencies(t *testing.T) {
	t.Parallel()

	all := []model.Module{
		{ID: "user-management", Path: "src/main/java/com/app/user_management"},
		{ID: "notification", Path: "src/main/java/com/app/notification"},
	}

	summaries := []model.JavaFileSummary{
		{
			Path:    "src/main/java/com/app/user_management/service/UserServiceImpl.java",
			Package: "com.app.user_management.service",
			Imports: []string{
				"com.app.notification.port.in.SendNotificationUseCase",
				"java.util.List",
			},
		},
	}

	deps := service.ResolveDependencies(summaries, all[0], all)
	require.Equal(t, []string{"notification"}, deps)
}

func TestResolveDependencies_SelfRemoved(t *testing.T) {
	t.Parallel()

	all := []model.Module{
		{ID: "user-management", Path: "src/main/java/com/app/user_management"},
	}

	summaries := []model.JavaFileSummary{
		{
			Path:    "src/main/java/com/app/user_management/service/UserServiceImpl.java",
			Package: "com.app.user_management.service",
			Imports: []string{
				"com.app.user_management.port.in.CreateUserUseCase",
			},
		},
	}

	deps := service.ResolveDependencies(summaries, all[0], all)
	require.Empty(t, deps)
}

func TestResolveDependencies_ExternalIgnored(t *testing.T) {
	t.Parallel()

	all := []model.Module{
		{ID: "user-management", Path: "src/main/java/com/app/user_management"},
	}

	summaries := []model.JavaFileSummary{
		{
			Path:    "src/main/java/com/app/user_management/service/UserServiceImpl.java",
			Package: "com.app.user_management.service",
			Imports: []string{
				"java.util.List",
				"org.springframework.stereotype.Service",
			},
		},
	}

	deps := service.ResolveDependencies(summaries, all[0], all)
	require.Empty(t, deps)
}
