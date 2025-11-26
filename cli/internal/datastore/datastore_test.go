package datastore

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/drata/drata-agent-cli/internal/config"
)

func TestNewDataStore(t *testing.T) {
	ds, err := New()
	if err != nil {
		t.Fatalf("failed to create data store: %v", err)
	}

	if ds == nil {
		t.Fatal("data store is nil")
	}
}

func TestDataStoreSetGet(t *testing.T) {
	ds, err := New()
	if err != nil {
		t.Fatalf("failed to create data store: %v", err)
	}

	// Test UUID
	uuid := "test-uuid-123"
	if err := ds.SetUUID(uuid); err != nil {
		t.Fatalf("failed to set UUID: %v", err)
	}
	if got := ds.GetUUID(); got != uuid {
		t.Errorf("expected UUID %s, got %s", uuid, got)
	}

	// Test AppVersion
	version := "1.0.0-test"
	if err := ds.SetAppVersion(version); err != nil {
		t.Fatalf("failed to set app version: %v", err)
	}
	if got := ds.GetAppVersion(); got != version {
		t.Errorf("expected version %s, got %s", version, got)
	}

	// Test AccessToken
	token := "test-token"
	if err := ds.SetAccessToken(token); err != nil {
		t.Fatalf("failed to set access token: %v", err)
	}
	if got := ds.GetAccessToken(); got != token {
		t.Errorf("expected token %s, got %s", token, got)
	}

	// Test SyncState
	state := SyncStateSuccess
	if err := ds.SetSyncState(state); err != nil {
		t.Fatalf("failed to set sync state: %v", err)
	}
	if got := ds.GetSyncState(); got != state {
		t.Errorf("expected state %s, got %s", state, got)
	}

	// Test Region
	region := config.RegionEU
	if err := ds.SetRegion(region); err != nil {
		t.Fatalf("failed to set region: %v", err)
	}
	if got := ds.GetRegion(); got != region {
		t.Errorf("expected region %s, got %s", region, got)
	}

	// Clean up
	ds.Clear()
}

func TestIsRegistered(t *testing.T) {
	ds, err := New()
	if err != nil {
		t.Fatalf("failed to create data store: %v", err)
	}
	ds.Clear()

	// Not registered initially
	if ds.IsRegistered() {
		t.Error("expected IsRegistered to be false initially")
	}

	// Set token
	if err := ds.SetAccessToken("test-token"); err != nil {
		t.Fatalf("failed to set access token: %v", err)
	}

	// Should be registered now
	if !ds.IsRegistered() {
		t.Error("expected IsRegistered to be true after setting token")
	}

	ds.Clear()
}

func TestUser(t *testing.T) {
	ds, err := New()
	if err != nil {
		t.Fatalf("failed to create data store: %v", err)
	}

	user := &User{
		ID:        1,
		EntryID:   "entry-123",
		Email:     "test@example.com",
		FirstName: "Test",
		LastName:  "User",
		Roles:     []string{"admin"},
	}

	if err := ds.SetUser(user); err != nil {
		t.Fatalf("failed to set user: %v", err)
	}

	got := ds.GetUser()
	if got == nil {
		t.Fatal("got nil user")
	}

	if got.Email != user.Email {
		t.Errorf("expected email %s, got %s", user.Email, got.Email)
	}
	if got.FirstName != user.FirstName {
		t.Errorf("expected first name %s, got %s", user.FirstName, got.FirstName)
	}

	ds.Clear()
}

func TestTimeSinceLastAttempt(t *testing.T) {
	ds, err := New()
	if err != nil {
		t.Fatalf("failed to create data store: %v", err)
	}
	ds.Clear()

	// No last attempt
	if mins := ds.MinutesSinceLastAttempt(); mins != -1 {
		t.Errorf("expected -1 for no last attempt, got %d", mins)
	}

	// Set last attempt to 5 minutes ago
	fiveMinutesAgo := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
	if err := ds.SetLastSyncAttemptedAt(fiveMinutesAgo); err != nil {
		t.Fatalf("failed to set last sync attempted: %v", err)
	}

	mins := ds.MinutesSinceLastAttempt()
	if mins < 4 || mins > 6 {
		t.Errorf("expected ~5 minutes since last attempt, got %d", mins)
	}

	ds.Clear()
}

func TestClear(t *testing.T) {
	ds, err := New()
	if err != nil {
		t.Fatalf("failed to create data store: %v", err)
	}

	// Set some data
	ds.SetAccessToken("token")
	ds.SetUUID("uuid")
	ds.SetAppVersion("1.0.0")

	// Clear
	if err := ds.Clear(); err != nil {
		t.Fatalf("failed to clear: %v", err)
	}

	// Verify cleared
	if ds.GetAccessToken() != "" {
		t.Error("access token not cleared")
	}
	if ds.GetUUID() != "" {
		t.Error("UUID not cleared")
	}
	if ds.GetAppVersion() != "" {
		t.Error("app version not cleared")
	}
}

func TestDataStorePersistence(t *testing.T) {
	ds1, err := New()
	if err != nil {
		t.Fatalf("failed to create data store: %v", err)
	}
	ds1.Clear()

	// Set data
	uuid := "persistence-test-uuid"
	if err := ds1.SetUUID(uuid); err != nil {
		t.Fatalf("failed to set UUID: %v", err)
	}

	// Create new data store instance (should load from file)
	ds2, err := New()
	if err != nil {
		t.Fatalf("failed to create second data store: %v", err)
	}

	// Verify data persisted
	if got := ds2.GetUUID(); got != uuid {
		t.Errorf("expected persisted UUID %s, got %s", uuid, got)
	}

	ds2.Clear()
}

func TestUpdate(t *testing.T) {
	ds, err := New()
	if err != nil {
		t.Fatalf("failed to create data store: %v", err)
	}
	ds.Clear()

	updates := map[string]interface{}{
		"uuid":       "update-test-uuid",
		"appVersion": "2.0.0",
		"syncState":  SyncStateSuccess,
		"region":     config.RegionAPAC,
	}

	if err := ds.Update(updates); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	if ds.GetUUID() != "update-test-uuid" {
		t.Error("UUID not updated")
	}
	if ds.GetAppVersion() != "2.0.0" {
		t.Error("app version not updated")
	}
	if ds.GetSyncState() != SyncStateSuccess {
		t.Error("sync state not updated")
	}
	if ds.GetRegion() != config.RegionAPAC {
		t.Error("region not updated")
	}

	ds.Clear()
}

func TestDataStoreFile(t *testing.T) {
	dataDir, err := config.GetDataDir()
	if err != nil {
		t.Fatalf("failed to get data dir: %v", err)
	}

	path := filepath.Join(dataDir, "app-data.json")

	ds, err := New()
	if err != nil {
		t.Fatalf("failed to create data store: %v", err)
	}
	ds.SetUUID("file-test")

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("data file not created")
	}

	ds.Clear()
}
