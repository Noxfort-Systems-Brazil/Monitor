// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/user_repo.go
// Author: Gabriel Moraes
// Date: 2026-09-03
// Modified: 2026-09-04 (PostgreSQL & Hot-Reload Support)

package storage

import (
	"database/sql"
	"fmt"
	"sync"

	"noxfort-monitor-server/internal/domain"
)

// UserRepositorySQLite implements domain.UserRepository with multi-database support.
type UserRepositorySQLite struct {
	mu     sync.RWMutex
	db     *sql.DB
	driver string
}

// NewUserRepository creates a new UserRepositorySQLite instance.
func NewUserRepository(db *sql.DB) *UserRepositorySQLite {
	return &UserRepositorySQLite{db: db, driver: "sqlite"}
}

// SetDB updates the database connection and dialect at runtime.
func (r *UserRepositorySQLite) SetDB(db *sql.DB, driver string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.db = db
	r.driver = driver
}

func (r *UserRepositorySQLite) getDB() (*sql.DB, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.db, r.driver
}

// GetByUsername retrieves a user by username.
func (r *UserRepositorySQLite) GetByUsername(username string) (*domain.User, error) {
	db, driver := r.getDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?;`, driver)
	row := db.QueryRow(query, username)

	var u domain.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query user by username: %w", err)
	}

	return &u, nil
}

// List returns all registered users.
func (r *UserRepositorySQLite) List() ([]*domain.User, error) {
	db, _ := r.getDB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `SELECT id, username, password_hash, role, created_at FROM users ORDER BY id ASC;`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, &u)
	}

	return users, nil
}

// Create inserts a new user record.
func (r *UserRepositorySQLite) Create(user *domain.User) error {
	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	if driver == "postgres" {
		query := `INSERT INTO users (username, password_hash, role, created_at) VALUES ($1, $2, $3, $4) RETURNING id;`
		err := db.QueryRow(query, user.Username, user.PasswordHash, user.Role, user.CreatedAt).Scan(&user.ID)
		if err != nil {
			return fmt.Errorf("failed to insert user into postgres: %w", err)
		}
		return nil
	}

	query := `INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, ?);`
	res, err := db.Exec(query, user.Username, user.PasswordHash, user.Role, user.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		user.ID = id
	}

	return nil
}

// Delete removes a user by ID.
func (r *UserRepositorySQLite) Delete(id int64) error {
	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`DELETE FROM users WHERE id = ?;`, driver)
	if _, err := db.Exec(query, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// DeleteByUsername removes a user by username.
func (r *UserRepositorySQLite) DeleteByUsername(username string) error {
	db, driver := r.getDB()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := AdaptQuery(`DELETE FROM users WHERE username = ?;`, driver)
	if _, err := db.Exec(query, username); err != nil {
		return fmt.Errorf("failed to delete user by username: %w", err)
	}
	return nil
}

// CountAdmins returns the total number of users with the ADMIN role.
func (r *UserRepositorySQLite) CountAdmins() (int, error) {
	db, _ := r.getDB()
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	query := `SELECT COUNT(*) FROM users WHERE role = 'ADMIN';`
	var count int
	if err := db.QueryRow(query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count admins: %w", err)
	}
	return count, nil
}
