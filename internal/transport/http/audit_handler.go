// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// File: internal/transport/http/audit_handler.go
// Author: Gabriel Moraes
// Date: 2026-09-04

package http

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"noxfort-monitor-server/internal/appdir"
	"noxfort-monitor-server/internal/domain"
)

// AuditHandler serves audit trail pages and JSON logs.
type AuditHandler struct {
	auditRepo domain.AuditRepository
}

// NewAuditHandler creates an instance of AuditHandler.
func NewAuditHandler(auditRepo domain.AuditRepository) *AuditHandler {
	return &AuditHandler{auditRepo: auditRepo}
}

// ServePage renders the comprehensive audit trail dashboard.
func (h *AuditHandler) ServePage(w http.ResponseWriter, r *http.Request) {
	var secLogs []domain.SecurityAuditLog
	var alertLogs []domain.AlertDispatchLog
	var transLogs []domain.DeviceStateTransition

	if h.auditRepo != nil {
		secLogs, _ = h.auditRepo.GetRecentSecurityAuditLogs(100)
		alertLogs, _ = h.auditRepo.GetRecentAlertDispatchLogs(100)
		transLogs, _ = h.auditRepo.GetRecentDeviceStateTransitions(100)
	}

	tmpl, err := template.ParseFiles(
		appdir.Path("web/templates/layout.html"),
		appdir.Path("web/templates/audit.html"),
	)
	if err != nil {
		log.Printf("[AUDIT] Template error: %v", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":          "Audit Trail",
		"SecurityLogs":   secLogs,
		"AlertLogs":      alertLogs,
		"TransitionLogs": transLogs,
	}

	_ = tmpl.Execute(w, data)
}

// HandleSecurityLogs returns recent security audit entries in JSON.
func (h *AuditHandler) HandleSecurityLogs(w http.ResponseWriter, r *http.Request) {
	if h.auditRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]domain.SecurityAuditLog{})
		return
	}
	logs, err := h.auditRepo.GetRecentSecurityAuditLogs(100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(logs)
}

// HandleAlertLogs returns recent alert dispatch entries in JSON.
func (h *AuditHandler) HandleAlertLogs(w http.ResponseWriter, r *http.Request) {
	if h.auditRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]domain.AlertDispatchLog{})
		return
	}
	logs, err := h.auditRepo.GetRecentAlertDispatchLogs(100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(logs)
}

// HandleTransitionLogs returns recent watchdog device state transitions in JSON.
func (h *AuditHandler) HandleTransitionLogs(w http.ResponseWriter, r *http.Request) {
	if h.auditRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]domain.DeviceStateTransition{})
		return
	}
	logs, err := h.auditRepo.GetRecentDeviceStateTransitions(100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(logs)
}
