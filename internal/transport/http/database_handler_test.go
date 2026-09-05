// Noxfort Monitor™ is an open-source industrial telemetry, observability, and incident response orchestration system.
// Copyright (C) 2026 Gabriel Moraes - Noxfort Systems

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"noxfort-monitor-server/internal/domain"
	"noxfort-monitor-server/internal/storage"
)

func TestDatabaseHandler_Endpoints(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "handler_test.db")

	cfg := domain.DatabaseConfig{
		Type:     "sqlite",
		FilePath: dbPath,
	}

	db, driver, err := storage.OpenConnection(cfg)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	_ = storage.InitPostgresSchema(db, "schema_monitor") // will be ignored for sqlite or error handled
	dbManager := storage.NewDBManager(db, driver, cfg)
	auditRepo := storage.NewAuditRepository(db, driver)
	handler := NewDatabaseHandler(dbManager, auditRepo)

	// 1. GET /api/settings/database/status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/settings/database/status", nil)
	recStatus := httptest.NewRecorder()
	handler.HandleStatus(recStatus, reqStatus)

	if recStatus.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on HandleStatus, got %d", recStatus.Code)
	}

	var statusResp map[string]interface{}
	if err := json.NewDecoder(recStatus.Body).Decode(&statusResp); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}
	if statusResp["status"] == nil {
		t.Errorf("Expected status object in response")
	}

	// 2. POST /api/settings/database/test (Testing SQLite)
	testBody := `{"type":"sqlite","file_path":"` + dbPath + `"}`
	reqTest := httptest.NewRequest(http.MethodPost, "/api/settings/database/test", strings.NewReader(testBody))
	reqTest.Header.Set("Content-Type", "application/json")
	recTest := httptest.NewRecorder()
	handler.HandleTest(recTest, reqTest)

	if recTest.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on HandleTest, got %d: %s", recTest.Code, recTest.Body.String())
	}

	var testResp map[string]interface{}
	if err := json.NewDecoder(recTest.Body).Decode(&testResp); err != nil {
		t.Fatalf("Failed to decode test response: %v", err)
	}
	if testResp["success"] != true {
		t.Errorf("Expected success to be true, got %v", testResp["success"])
	}

	// 3. POST /api/settings/database/save (Saving SQLite)
	dbPath2 := filepath.Join(tempDir, "handler_test_save.db")
	saveBody := `{"type":"sqlite","file_path":"` + dbPath2 + `","migrate":false}`
	reqSave := httptest.NewRequest(http.MethodPost, "/api/settings/database/save", strings.NewReader(saveBody))
	reqSave.Header.Set("Content-Type", "application/json")
	recSave := httptest.NewRecorder()
	handler.HandleSave(recSave, reqSave)

	if recSave.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on HandleSave, got %d: %s", recSave.Code, recSave.Body.String())
	}

	// 4. GET /server (ServePage)
	reqPage := httptest.NewRequest(http.MethodGet, "/server", nil)
	recPage := httptest.NewRecorder()
	handler.ServePage(recPage, reqPage)

	if recPage.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on ServePage, got %d: %s", recPage.Code, recPage.Body.String())
	}
	if !strings.Contains(recPage.Body.String(), "dbSettingsForm") {
		t.Errorf("Expected ServePage output to contain dbSettingsForm")
	}
}

type mockDBService struct {
	status   domain.DatabaseStatus
	cfg      domain.DatabaseConfig
	testErr  error
	switchErr error
}

func (m *mockDBService) GetStatus() domain.DatabaseStatus {
	return m.status
}

func (m *mockDBService) GetConfig() domain.DatabaseConfig {
	return m.cfg
}

func (m *mockDBService) TestConnection(cfg domain.DatabaseConfig) (domain.DatabaseStatus, error) {
	return m.status, m.testErr
}

func (m *mockDBService) Switch(newCfg domain.DatabaseConfig, migrate bool) error {
	m.cfg = newCfg
	return m.switchErr
}

type mockProvisioner struct {
	provisionErr error
	called       bool
}

func (m *mockProvisioner) ProvisionUser(host string, port int, dbname, adminUser, adminPassword, newUser, newPassword, sslmode string) error {
	m.called = true
	return m.provisionErr
}

func TestDatabaseHandler_MockService(t *testing.T) {
	mockSvc := &mockDBService{
		status: domain.DatabaseStatus{
			Connected:    true,
			Type:         "postgres",
			SchemaExists: true,
		},
		cfg: domain.DatabaseConfig{
			Type:   "postgres",
			Host:   "localhost",
			DBName: "testdb",
		},
	}
	mockProv := &mockProvisioner{}

	handler := NewDatabaseHandlerWithProvisioner(mockSvc, nil, mockProv)

	// Test Status
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/settings/database/status", nil)
	recStatus := httptest.NewRecorder()
	handler.HandleStatus(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", recStatus.Code)
	}

	// Test Provision User
	provBody := `{"host":"localhost","port":5432,"dbname":"testdb","new_user":"testuser","new_password":"pwd","admin_user":"postgres","admin_password":"pwd"}`
	reqProv := httptest.NewRequest(http.MethodPost, "/api/settings/database/provision-user", strings.NewReader(provBody))
	reqProv.Header.Set("Content-Type", "application/json")
	recProv := httptest.NewRecorder()
	handler.HandleProvisionUser(recProv, reqProv)

	if recProv.Code != http.StatusOK {
		t.Errorf("Expected 200 on provision-user, got %d: %s", recProv.Code, recProv.Body.String())
	}
	if !mockProv.called {
		t.Errorf("Expected provisioner to be called")
	}
}

