// Package datastore provides persistent storage for the Drata Agent CLI.
package datastore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/drata/drata-agent-cli/internal/config"
)

// SyncState represents the state of the last sync operation.
type SyncState string

const (
	SyncStateSuccess SyncState = "SUCCESS"
	SyncStateError   SyncState = "ERROR"
	SyncStateRunning SyncState = "RUNNING"
	SyncStateUnknown SyncState = "UNKNOWN"
)

// User represents the authenticated user information.
type User struct {
	ID                 int      `json:"id"`
	EntryID            string   `json:"entryId"`
	Email              string   `json:"email"`
	FirstName          string   `json:"firstName"`
	LastName           string   `json:"lastName"`
	JobTitle           string   `json:"jobTitle,omitempty"`
	AvatarURL          string   `json:"avatarUrl"`
	Roles              []string `json:"roles"`
	DrataTermsAgreedAt string   `json:"drataTermsAgreedAt"`
	CreatedAt          string   `json:"createdAt"`
	UpdatedAt          string   `json:"updatedAt"`
	Signature          string   `json:"signature"`
	Language           string   `json:"language"`
}

// DataStore holds all persistent data for the agent.
type DataStore struct {
	UUID                   string        `json:"uuid,omitempty"`
	AppVersion             string        `json:"appVersion,omitempty"`
	AccessToken            string        `json:"accessToken,omitempty"`
	User                   *User         `json:"user,omitempty"`
	SyncState              SyncState     `json:"syncState,omitempty"`
	LastCheckedAt          string        `json:"lastCheckedAt,omitempty"`
	LastSyncAttemptedAt    string        `json:"lastSyncAttemptedAt,omitempty"`
	ComplianceData         interface{}   `json:"complianceData,omitempty"`
	WinAvServicesMatchList []string      `json:"winAvServicesMatchList,omitempty"`
	Region                 config.Region `json:"region,omitempty"`

	mu   sync.RWMutex
	path string
}

// New creates a new DataStore instance.
func New() (*DataStore, error) {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}

	path := filepath.Join(dataDir, "app-data.json")
	ds := &DataStore{
		path: path,
	}

	// Load existing data if file exists
	if _, err := os.Stat(path); err == nil {
		if err := ds.load(); err != nil {
			return nil, fmt.Errorf("failed to load data store: %w", err)
		}
	}

	return ds, nil
}

// load reads the data store from disk.
func (ds *DataStore) load() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	data, err := os.ReadFile(ds.path)
	if err != nil {
		return err
	}

	// Preserve the path
	path := ds.path
	if err := json.Unmarshal(data, ds); err != nil {
		return err
	}
	ds.path = path

	return nil
}

// save writes the data store to disk.
func (ds *DataStore) save() error {
	data, err := json.MarshalIndent(ds, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(ds.path, data, 0600)
}

// GetAccessToken returns the access token.
func (ds *DataStore) GetAccessToken() string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.AccessToken
}

// SetAccessToken sets the access token.
func (ds *DataStore) SetAccessToken(token string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.AccessToken = token
	return ds.save()
}

// GetUser returns the user information.
func (ds *DataStore) GetUser() *User {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.User
}

// SetUser sets the user information.
func (ds *DataStore) SetUser(user *User) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.User = user
	return ds.save()
}

// GetSyncState returns the sync state.
func (ds *DataStore) GetSyncState() SyncState {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.SyncState
}

// SetSyncState sets the sync state.
func (ds *DataStore) SetSyncState(state SyncState) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.SyncState = state
	return ds.save()
}

// GetLastCheckedAt returns the last checked timestamp.
func (ds *DataStore) GetLastCheckedAt() string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.LastCheckedAt
}

// SetLastCheckedAt sets the last checked timestamp.
func (ds *DataStore) SetLastCheckedAt(timestamp string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.LastCheckedAt = timestamp
	return ds.save()
}

// GetLastSyncAttemptedAt returns the last sync attempted timestamp.
func (ds *DataStore) GetLastSyncAttemptedAt() string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.LastSyncAttemptedAt
}

// SetLastSyncAttemptedAt sets the last sync attempted timestamp.
func (ds *DataStore) SetLastSyncAttemptedAt(timestamp string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.LastSyncAttemptedAt = timestamp
	return ds.save()
}

// GetRegion returns the region.
func (ds *DataStore) GetRegion() config.Region {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.Region
}

// SetRegion sets the region.
func (ds *DataStore) SetRegion(region config.Region) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.Region = region
	return ds.save()
}

// GetUUID returns the UUID.
func (ds *DataStore) GetUUID() string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.UUID
}

// SetUUID sets the UUID.
func (ds *DataStore) SetUUID(uuid string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.UUID = uuid
	return ds.save()
}

// GetAppVersion returns the app version.
func (ds *DataStore) GetAppVersion() string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.AppVersion
}

// SetAppVersion sets the app version.
func (ds *DataStore) SetAppVersion(version string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.AppVersion = version
	return ds.save()
}

// GetComplianceData returns the compliance data.
func (ds *DataStore) GetComplianceData() interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.ComplianceData
}

// SetComplianceData sets the compliance data.
func (ds *DataStore) SetComplianceData(data interface{}) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.ComplianceData = data
	return ds.save()
}

// GetWinAvServicesMatchList returns the Windows AV services match list.
func (ds *DataStore) GetWinAvServicesMatchList() []string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.WinAvServicesMatchList
}

// SetWinAvServicesMatchList sets the Windows AV services match list.
func (ds *DataStore) SetWinAvServicesMatchList(list []string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.WinAvServicesMatchList = list
	return ds.save()
}

// IsRegistered returns true if the agent is registered.
func (ds *DataStore) IsRegistered() bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.AccessToken != ""
}

// IsInitDataReady returns true if initialization data is ready.
func (ds *DataStore) IsInitDataReady() bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.WinAvServicesMatchList != nil
}

// Clear clears all data from the store.
func (ds *DataStore) Clear() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	path := ds.path
	mu := ds.mu

	// Reset all fields except path and mutex
	ds.UUID = ""
	ds.AppVersion = ""
	ds.AccessToken = ""
	ds.User = nil
	ds.SyncState = ""
	ds.LastCheckedAt = ""
	ds.LastSyncAttemptedAt = ""
	ds.ComplianceData = nil
	ds.WinAvServicesMatchList = nil
	ds.Region = ""
	ds.path = path
	ds.mu = mu

	return ds.save()
}

// Update updates multiple fields at once.
func (ds *DataStore) Update(updates map[string]interface{}) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	for key, value := range updates {
		switch key {
		case "uuid":
			if v, ok := value.(string); ok {
				ds.UUID = v
			}
		case "appVersion":
			if v, ok := value.(string); ok {
				ds.AppVersion = v
			}
		case "accessToken":
			if v, ok := value.(string); ok {
				ds.AccessToken = v
			}
		case "syncState":
			if v, ok := value.(SyncState); ok {
				ds.SyncState = v
			}
		case "lastCheckedAt":
			if v, ok := value.(string); ok {
				ds.LastCheckedAt = v
			}
		case "lastSyncAttemptedAt":
			if v, ok := value.(string); ok {
				ds.LastSyncAttemptedAt = v
			}
		case "complianceData":
			ds.ComplianceData = value
		case "winAvServicesMatchList":
			if v, ok := value.([]string); ok {
				ds.WinAvServicesMatchList = v
			}
		case "region":
			if v, ok := value.(config.Region); ok {
				ds.Region = v
			}
		case "user":
			if v, ok := value.(*User); ok {
				ds.User = v
			}
		}
	}

	return ds.save()
}

// MinutesSinceLastAttempt returns the minutes since the last sync attempt.
func (ds *DataStore) MinutesSinceLastAttempt() int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.LastSyncAttemptedAt == "" {
		return -1
	}

	lastAttempt, err := time.Parse(time.RFC3339, ds.LastSyncAttemptedAt)
	if err != nil {
		return -1
	}

	return int(time.Since(lastAttempt).Minutes())
}

// HoursSinceLastSuccess returns the hours since the last successful sync.
func (ds *DataStore) HoursSinceLastSuccess() int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if ds.LastCheckedAt == "" {
		return -1
	}

	lastSuccess, err := time.Parse(time.RFC3339, ds.LastCheckedAt)
	if err != nil {
		return -1
	}

	return int(time.Since(lastSuccess).Hours())
}
