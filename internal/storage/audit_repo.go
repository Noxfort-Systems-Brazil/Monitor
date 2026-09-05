// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/storage/audit_repo.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package storage

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"noxfort-monitor-server/internal/domain"
)

// AuditRepositorySQL implements domain.AuditRepository with dynamic dialect adaptation.
type AuditRepositorySQL struct {
	mu     sync.RWMutex
	db     *sql.DB
	driver string
}

// NewAuditRepository creates a new audit repository instance.
func NewAuditRepository(db *sql.DB, driver string) *AuditRepositorySQL {
	if driver == "" {
		driver = "sqlite"
	}
	return &AuditRepositorySQL{
		db:     db,
		driver: driver,
	}
}

// SetDB dynamically reconfigures the active database connection.
func (r *AuditRepositorySQL) SetDB(db *sql.DB, driver string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.db = db
	r.driver = driver
}

// SaveSecurityAuditLog records an authentication or sensitive administrative event.
func (r *AuditRepositorySQL) SaveSecurityAuditLog(entry *domain.SecurityAuditLog) error {
	r.mu.RLock()
	db := r.db
	driver := r.driver
	r.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	query := AdaptQuery(`
		INSERT INTO security_audit_logs (created_at, username, action, details, ip_address)
		VALUES (?, ?, ?, ?, ?);
	`, driver)

	_, err := db.Exec(query, entry.CreatedAt, entry.Username, entry.Action, entry.Details, entry.IPAddress)
	if err != nil {
		return fmt.Errorf("failed to insert security audit log: %w", err)
	}
	return nil
}

// GetRecentSecurityAuditLogs queries the latest N security audit logs.
func (r *AuditRepositorySQL) GetRecentSecurityAuditLogs(limit int) ([]domain.SecurityAuditLog, error) {
	r.mu.RLock()
	db := r.db
	driver := r.driver
	r.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if limit <= 0 {
		limit = 50
	}

	query := AdaptQuery(`
		SELECT id, created_at, username, action, COALESCE(details, ''), COALESCE(ip_address, '')
		FROM security_audit_logs
		ORDER BY created_at DESC
		LIMIT ?;
	`, driver)

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query security audit logs: %w", err)
	}
	defer rows.Close()

	var logs []domain.SecurityAuditLog
	for rows.Next() {
		var l domain.SecurityAuditLog
		if err := rows.Scan(&l.ID, &l.CreatedAt, &l.Username, &l.Action, &l.Details, &l.IPAddress); err != nil {
			return nil, fmt.Errorf("failed to scan security audit log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// SaveAlertDispatchLog registers an alert notification delivery attempt.
func (r *AuditRepositorySQL) SaveAlertDispatchLog(entry *domain.AlertDispatchLog) error {
	r.mu.RLock()
	db := r.db
	driver := r.driver
	r.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	if entry.DispatchedAt.IsZero() {
		entry.DispatchedAt = time.Now()
	}

	query := AdaptQuery(`
		INSERT INTO alert_dispatch_logs (telemetry_id, channel, recipient, role, status, error_reason, dispatched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`, driver)

	_, err := db.Exec(query,
		entry.TelemetryID,
		entry.Channel,
		entry.Recipient,
		entry.Role,
		entry.Status,
		entry.ErrorReason,
		entry.DispatchedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert alert dispatch log: %w", err)
	}
	return nil
}

// GetRecentAlertDispatchLogs returns recent alert dispatch records.
func (r *AuditRepositorySQL) GetRecentAlertDispatchLogs(limit int) ([]domain.AlertDispatchLog, error) {
	r.mu.RLock()
	db := r.db
	driver := r.driver
	r.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if limit <= 0 {
		limit = 50
	}

	query := AdaptQuery(`
		SELECT id, telemetry_id, channel, recipient, COALESCE(role, ''), status, COALESCE(error_reason, ''), dispatched_at
		FROM alert_dispatch_logs
		ORDER BY dispatched_at DESC
		LIMIT ?;
	`, driver)

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query alert dispatch logs: %w", err)
	}
	defer rows.Close()

	var logs []domain.AlertDispatchLog
	for rows.Next() {
		var l domain.AlertDispatchLog
		if err := rows.Scan(&l.ID, &l.TelemetryID, &l.Channel, &l.Recipient, &l.Role, &l.Status, &l.ErrorReason, &l.DispatchedAt); err != nil {
			return nil, fmt.Errorf("failed to scan alert dispatch log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// SaveDeviceStateTransition records a watchdog health state change.
func (r *AuditRepositorySQL) SaveDeviceStateTransition(entry *domain.DeviceStateTransition) error {
	r.mu.RLock()
	db := r.db
	driver := r.driver
	r.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	if entry.TransitionAt.IsZero() {
		entry.TransitionAt = time.Now()
	}

	query := AdaptQuery(`
		INSERT INTO device_state_transitions (device_identifier, previous_state, new_state, duration_offline_sec, transition_at)
		VALUES (?, ?, ?, ?, ?);
	`, driver)

	_, err := db.Exec(query,
		entry.DeviceIdentifier,
		entry.PreviousState,
		entry.NewState,
		entry.DurationOfflineSec,
		entry.TransitionAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert device state transition: %w", err)
	}
	return nil
}

// GetRecentDeviceStateTransitions returns recent watchdog state transition events.
func (r *AuditRepositorySQL) GetRecentDeviceStateTransitions(limit int) ([]domain.DeviceStateTransition, error) {
	r.mu.RLock()
	db := r.db
	driver := r.driver
	r.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	if limit <= 0 {
		limit = 50
	}

	query := AdaptQuery(`
		SELECT id, device_identifier, COALESCE(previous_state, ''), new_state, duration_offline_sec, transition_at
		FROM device_state_transitions
		ORDER BY transition_at DESC
		LIMIT ?;
	`, driver)

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query device state transitions: %w", err)
	}
	defer rows.Close()

	var logs []domain.DeviceStateTransition
	for rows.Next() {
		var l domain.DeviceStateTransition
		if err := rows.Scan(&l.ID, &l.DeviceIdentifier, &l.PreviousState, &l.NewState, &l.DurationOfflineSec, &l.TransitionAt); err != nil {
			return nil, fmt.Errorf("failed to scan device state transition: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, nil
}
