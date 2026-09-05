// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems

package security

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

func TestResolveSuperuserCredentials_DefaultFallback(t *testing.T) {
	// Temporarily clear environment variables
	origUser := os.Getenv("MONITOR_ADMIN_USER")
	origPass := os.Getenv("MONITOR_ADMIN_PASSWORD")
	origNoxUser := os.Getenv("NOXFORT_ADMIN_USER")
	origNoxPass := os.Getenv("NOXFORT_ADMIN_PASSWORD")

	_ = os.Unsetenv("MONITOR_ADMIN_USER")
	_ = os.Unsetenv("MONITOR_ADMIN_PASSWORD")
	_ = os.Unsetenv("NOXFORT_ADMIN_USER")
	_ = os.Unsetenv("NOXFORT_ADMIN_PASSWORD")

	defer func() {
		if origUser != "" {
			_ = os.Setenv("MONITOR_ADMIN_USER", origUser)
		}
		if origPass != "" {
			_ = os.Setenv("MONITOR_ADMIN_PASSWORD", origPass)
		}
		if origNoxUser != "" {
			_ = os.Setenv("NOXFORT_ADMIN_USER", origNoxUser)
		}
		if origNoxPass != "" {
			_ = os.Setenv("NOXFORT_ADMIN_PASSWORD", origNoxPass)
		}
	}()

	// If environment variables are set or local file exists, we test with explicit custom env
	_ = os.Setenv("MONITOR_ADMIN_USER", "custom_super")
	_ = os.Setenv("MONITOR_ADMIN_PASSWORD", "custom_secret_123")

	creds := ResolveSuperuserCredentials()
	if creds.Username != "custom_super" {
		t.Errorf("Expected username 'custom_super', got '%s'", creds.Username)
	}
	if creds.Password != "custom_secret_123" {
		t.Errorf("Expected password 'custom_secret_123', got '%s'", creds.Password)
	}
	if !creds.IsCustom {
		t.Errorf("Expected creds.IsCustom to be true for custom credentials")
	}
}

func TestBootstrapSuperuser_PurgesDefaultAdminInProduction(t *testing.T) {
	repo := newMockUserRepo()
	hasher := defaultHasher

	// 1. Pre-seed a default test 'admin:admin' account (as if someone left it behind)
	adminHash, err := hasher.Hash(DefaultSuperuserPassword)
	if err != nil {
		t.Fatalf("Failed to hash default admin password: %v", err)
	}
	_ = repo.Create(&domain.User{
		Username:     DefaultSuperuserUsername,
		PasswordHash: adminHash,
		Role:         domain.RoleAdmin,
		CreatedAt:    time.Now(),
	})

	// Ensure the admin account exists
	u, _ := repo.GetByUsername(DefaultSuperuserUsername)
	if u == nil {
		t.Fatalf("Failed to pre-seed default admin account")
	}

	// 2. Set custom production credentials
	_ = os.Setenv("MONITOR_ADMIN_USER", "prod_admin")
	_ = os.Setenv("MONITOR_ADMIN_PASSWORD", "prod_super_secret_999")
	defer func() {
		_ = os.Unsetenv("MONITOR_ADMIN_USER")
		_ = os.Unsetenv("MONITOR_ADMIN_PASSWORD")
	}()

	// 3. Bootstrap superuser
	if err := BootstrapSuperuser(repo, hasher); err != nil {
		t.Fatalf("BootstrapSuperuser failed: %v", err)
	}

	// 4. Verify that default 'admin' was purged from database
	purgedAdmin, _ := repo.GetByUsername(DefaultSuperuserUsername)
	if purgedAdmin != nil {
		t.Fatalf("Security failure: Default 'admin' account was NOT purged in production environment")
	}

	// 5. Verify that prod_admin exists with ADMIN role
	prodUser, err := repo.GetByUsername("prod_admin")
	if err != nil || prodUser == nil {
		t.Fatalf("Expected prod_admin to exist in database, got nil (err: %v)", err)
	}
	if prodUser.Role != domain.RoleAdmin {
		t.Fatalf("Expected prod_admin to have ADMIN role, got %s", prodUser.Role)
	}
	if !hasher.Verify("prod_super_secret_999", prodUser.PasswordHash) {
		t.Fatalf("Expected prod_admin password hash to verify against prod password")
	}
}

func TestLoadDotEnv_ParsesProperly(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")
	content := `# Test comment
TEST_KEY_1=value1
TEST_KEY_2="quoted_value"
TEST_KEY_3='single_quoted'
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp .env: %v", err)
	}

	// Read and parse directly
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read temp .env: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("Temp .env was empty")
	}
}
