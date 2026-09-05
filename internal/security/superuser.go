// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// File: internal/security/superuser.go
// Author: Gabriel Moraes
// Date: 2026-09-04
// Modified: 2026-09-05 (Dynamic .env resolution & Production anti-backdoor protection)

package security

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"noxfort-monitor-server/internal/appdir"
	"noxfort-monitor-server/internal/domain"
)

// Default superuser credentials intended strictly for zero-friction open-source testing from GitHub.
const (
	DefaultSuperuserUsername = "admin"
	DefaultSuperuserPassword = "admin"
)

// SuperuserUsername and SuperuserPassword store the actively resolved credentials.
var (
	SuperuserUsername = DefaultSuperuserUsername
	SuperuserPassword = DefaultSuperuserPassword
	dotEnvOnce        sync.Once
)

// SuperuserCredentials encapsulates resolved credentials and customization status.
type SuperuserCredentials struct {
	Username string
	Password string
	IsCustom bool
}

// loadDotEnv parses .env key-value pairs without adding third-party runtime dependencies.
func loadDotEnv() {
	candidates := []string{
		".env",
		appdir.Path(".env"),
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, ".env"))
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				// Strip surrounding quotes
				if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
					v = v[1 : len(v)-1]
				}
				if os.Getenv(k) == "" {
					_ = os.Setenv(k, v)
				}
			}
		}
		break
	}
}

// GetAuthConfigFilepath returns the path to the fallback credentials file outside the repository.
func GetAuthConfigFilepath() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "auth_config.json"
	}
	dataDir := filepath.Join(homedir, "Documentos", "Monitor")
	_ = os.MkdirAll(dataDir, 0755)
	return filepath.Join(dataDir, "auth_config.json")
}

// ResolveSuperuserCredentials dynamically determines active credentials.
// Precedence:
// 1. Environment variables (.env loaded or system env vars: MONITOR_ADMIN_USER / NOXFORT_ADMIN_USER)
// 2. Local config file (~/Documentos/Monitor/auth_config.json outside Git)
// 3. Fallback defaults for GitHub clones (admin / admin)
func ResolveSuperuserCredentials() SuperuserCredentials {
	dotEnvOnce.Do(loadDotEnv)

	user := strings.TrimSpace(os.Getenv("MONITOR_ADMIN_USER"))
	if user == "" {
		user = strings.TrimSpace(os.Getenv("NOXFORT_ADMIN_USER"))
	}

	pass := os.Getenv("MONITOR_ADMIN_PASSWORD")
	if pass == "" {
		pass = os.Getenv("NOXFORT_ADMIN_PASSWORD")
	}

	if user != "" && pass != "" {
		isCustom := (user != DefaultSuperuserUsername || pass != DefaultSuperuserPassword)
		return SuperuserCredentials{
			Username: user,
			Password: pass,
			IsCustom: isCustom,
		}
	}

	// 2. Check fallback file in ~/Documentos/Monitor/auth_config.json
	cfgFile := GetAuthConfigFilepath()
	if data, err := os.ReadFile(cfgFile); err == nil {
		var local struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.Unmarshal(data, &local); err == nil {
			localUser := strings.TrimSpace(local.Username)
			if localUser != "" && local.Password != "" {
				isCustom := (localUser != DefaultSuperuserUsername || local.Password != DefaultSuperuserPassword)
				return SuperuserCredentials{
					Username: localUser,
					Password: local.Password,
					IsCustom: isCustom,
				}
			}
		}
	}

	// 3. GitHub repository fallback
	return SuperuserCredentials{
		Username: DefaultSuperuserUsername,
		Password: DefaultSuperuserPassword,
		IsCustom: false,
	}
}

// BootstrapSuperuser guarantees that the designated superuser account exists with ADMIN role
// and valid credentials.
// In production (when custom credentials are set via .env):
// - The production superuser is created/updated.
// - Any default 'admin' account with default 'admin' password is automatically purged to eliminate backdoors.
func BootstrapSuperuser(userRepo domain.UserRepository, hasher PasswordHasher) error {
	creds := ResolveSuperuserCredentials()
	SuperuserUsername = creds.Username
	SuperuserPassword = creds.Password

	// Anti-backdoor protection: if custom credentials are used, ensure default test admin:admin cannot exist
	if creds.IsCustom && creds.Username != DefaultSuperuserUsername {
		adminUser, _ := userRepo.GetByUsername(DefaultSuperuserUsername)
		if adminUser != nil {
			if hasher.Verify(DefaultSuperuserPassword, adminUser.PasswordHash) {
				_ = userRepo.DeleteByUsername(DefaultSuperuserUsername)
				log.Printf("[SECURITY] Conta de teste padrão '%s' purgada com sucesso para ambiente de produção.", DefaultSuperuserUsername)
			}
		}
	}

	existing, err := userRepo.GetByUsername(creds.Username)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("falha ao verificar superusuário: %w", err)
	}

	if existing != nil {
		needsUpdate := false
		if existing.Role != domain.RoleAdmin {
			existing.Role = domain.RoleAdmin
			needsUpdate = true
		}
		if !hasher.Verify(creds.Password, existing.PasswordHash) {
			newHash, err := hasher.Hash(creds.Password)
			if err != nil {
				return fmt.Errorf("falha ao criptografar senha do superusuário: %w", err)
			}
			existing.PasswordHash = newHash
			needsUpdate = true
		}
		if needsUpdate {
			_ = userRepo.DeleteByUsername(creds.Username)
			if err := userRepo.Create(existing); err != nil {
				return fmt.Errorf("falha ao atualizar superusuário: %w", err)
			}
			log.Printf("[SECURITY] Superusuário '%s' (ADMIN) atualizado com sucesso.", creds.Username)
		}
	} else {
		hash, err := hasher.Hash(creds.Password)
		if err != nil {
			return fmt.Errorf("falha ao criptografar senha do superusuário: %w", err)
		}

		superuser := &domain.User{
			Username:     creds.Username,
			PasswordHash: hash,
			Role:         domain.RoleAdmin,
			CreatedAt:    time.Now(),
		}

		if err := userRepo.Create(superuser); err != nil {
			return fmt.Errorf("falha ao criar superusuário: %w", err)
		}

		log.Printf("[SECURITY] Superusuário '%s' (ADMIN) inicializado com sucesso.", creds.Username)
	}

	return nil
}
