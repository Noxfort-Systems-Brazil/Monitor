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
// File: internal/domain/user.go
// Author: Gabriel Moraes
// Date: 2026-09-03

package domain

import "time"

// User roles for Noxfort Monitor
const (
	RoleAdmin    = "ADMIN"
	RoleOperator = "OPERATOR"
)

// User represents a system operator or administrator account.
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never expose hash in JSON
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserRepository defines the persistence contract for user accounts.
type UserRepository interface {
	GetByUsername(username string) (*User, error)
	List() ([]*User, error)
	Create(user *User) error
	Delete(id int64) error
	DeleteByUsername(username string) error
	CountAdmins() (int, error)
}
