// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems
//
// File: internal/storage/telemetry_repo_test.go

package storage

import (
	"fmt"
	"testing"
	"time"

	"noxfort-monitor-server/internal/domain"
)

func TestTelemetryRepository_SingleAndBatchInsert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTelemetryRepository(db)
	now := time.Now().Truncate(time.Second)

	// 1. Test single insert
	singleEvent := &domain.IncomingEvent{
		Category:   domain.CategoryHardware,
		Origin:     "PLC-01",
		Level:      domain.LevelWarning,
		Message:    "Motor temperature high",
		OccurredAt: now,
	}

	if err := repo.SaveEvent("PLC-01", singleEvent); err != nil {
		t.Fatalf("SaveEvent failed: %v", err)
	}

	incidents, err := repo.GetRecentIncidents(10)
	if err != nil {
		t.Fatalf("GetRecentIncidents failed: %v", err)
	}
	if len(incidents) != 1 {
		t.Fatalf("Expected 1 incident, got %d", len(incidents))
	}
	if incidents[0].Origin != "PLC-01" || incidents[0].Level != domain.LevelWarning {
		t.Errorf("Unexpected incident data: %+v", incidents[0])
	}

	// 2. Test batch insert
	records := make([]TelemetryRecord, 0, 5)
	for i := 1; i <= 5; i++ {
		records = append(records, TelemetryRecord{
			Identifier: "PLC-02",
			Event: &domain.IncomingEvent{
				Category:   domain.CategorySoftware,
				Origin:     "PLC-02",
				Level:      domain.LevelCritical,
				Message:    fmt.Sprintf("Critical failure code #%d", i),
				OccurredAt: now.Add(time.Duration(i) * time.Minute),
			},
		})
	}

	if err := repo.SaveEventsBatch(records); err != nil {
		t.Fatalf("SaveEventsBatch failed: %v", err)
	}

	// Total should be 1 + 5 = 6
	incidents, err = repo.GetRecentIncidents(10)
	if err != nil {
		t.Fatalf("GetRecentIncidents failed: %v", err)
	}
	if len(incidents) != 6 {
		t.Fatalf("Expected 6 incidents, got %d", len(incidents))
	}

	// 3. Test GetRecentIncidentsByDevice
	plc02Incidents, err := repo.GetRecentIncidentsByDevice("PLC-02", 10)
	if err != nil {
		t.Fatalf("GetRecentIncidentsByDevice failed: %v", err)
	}
	if len(plc02Incidents) != 5 {
		t.Fatalf("Expected 5 incidents for PLC-02, got %d", len(plc02Incidents))
	}

	plc01Incidents, err := repo.GetRecentIncidentsByDevice("PLC-01", 10)
	if err != nil {
		t.Fatalf("GetRecentIncidentsByDevice for PLC-01 failed: %v", err)
	}
	if len(plc01Incidents) != 1 {
		t.Fatalf("Expected 1 incident for PLC-01, got %d", len(plc01Incidents))
	}
}

func TestBufferedTelemetryWriter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTelemetryRepository(db)
	writer := NewBufferedTelemetryWriter(repo, 10, 50*time.Millisecond)
	now := time.Now().Truncate(time.Second)

	// Enqueue 15 events (exceeds batchSize of 10)
	for i := 1; i <= 15; i++ {
		writer.Enqueue("ROBOT-ARM-01", &domain.IncomingEvent{
			Category:   domain.CategoryHardware,
			Origin:     "ROBOT-ARM-01",
			Level:      domain.LevelInfo,
			Message:    fmt.Sprintf("Position check %d", i),
			OccurredAt: now.Add(time.Duration(i) * time.Second),
		})
	}

	// Close cleanly flushes remaining 5 items in buffer
	writer.Close()

	incidents, err := repo.GetRecentIncidents(50)
	if err != nil {
		t.Fatalf("GetRecentIncidents failed: %v", err)
	}
	if len(incidents) != 15 {
		t.Fatalf("Expected 15 incidents saved via buffer, got %d", len(incidents))
	}
}
