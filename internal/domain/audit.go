// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/domain/audit.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package domain

import "time"

// SecurityAuditLog records access control and sensitive operational events.
// Mirrored after Synapse and Carina audit models.
type SecurityAuditLog struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username"`
	Action    string    `json:"action"` // e.g. "AUTH_LOGIN_SUCCESS", "AUTH_LOGIN_FAILED", "CONFIG_UPDATE", "DEVICE_DELETE"
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
}

// AlertDispatchLog records incident delivery for SLA, audit compliance, and delivery verification.
type AlertDispatchLog struct {
	ID           int64     `json:"id"`
	TelemetryID  *int64    `json:"telemetry_id,omitempty"`
	Channel      string    `json:"channel"`   // "EMAIL", "TELEGRAM"
	Recipient    string    `json:"recipient"` // email or telegram chat_id
	Role         string    `json:"role"`
	Status       string    `json:"status"` // "SENT", "FAILED", "SKIPPED"
	ErrorReason  string    `json:"error_reason,omitempty"`
	DispatchedAt time.Time `json:"dispatched_at"`
}

// DeviceStateTransition records watchdog availability transitions (ONLINE/OFFLINE/RECOVERY).
type DeviceStateTransition struct {
	ID                 int64     `json:"id"`
	DeviceIdentifier   string    `json:"device_identifier"`
	PreviousState      string    `json:"previous_state"`
	NewState           string    `json:"new_state"` // "ONLINE", "OFFLINE", "DEGRADED"
	DurationOfflineSec int64     `json:"duration_offline_sec"`
	TransitionAt       time.Time `json:"transition_at"`
}

// AuditRepository defines persistence contracts for auditing subsystems.
type AuditRepository interface {
	SaveSecurityAuditLog(log *SecurityAuditLog) error
	GetRecentSecurityAuditLogs(limit int) ([]SecurityAuditLog, error)

	SaveAlertDispatchLog(log *AlertDispatchLog) error
	GetRecentAlertDispatchLogs(limit int) ([]AlertDispatchLog, error)

	SaveDeviceStateTransition(log *DeviceStateTransition) error
	GetRecentDeviceStateTransitions(limit int) ([]DeviceStateTransition, error)
}
