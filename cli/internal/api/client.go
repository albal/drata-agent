// Package api provides the HTTP client for communicating with the Drata API.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/drata/drata-agent-cli/internal/config"
	"github.com/drata/drata-agent-cli/internal/datastore"
	"github.com/drata/drata-agent-cli/internal/osquery"
)

const (
	requestTimeout = 5 * time.Minute
)

// AuthResponse represents the response from authentication endpoints.
type AuthResponse struct {
	AccessToken string `json:"accessToken,omitempty"`
}

// MeResponse represents the user information response.
type MeResponse struct {
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

// AgentV2Response represents the response from agent endpoints.
type AgentV2Response struct {
	LastCheckedAt          string      `json:"lastcheckedAt,omitempty"`
	Data                   interface{} `json:"data,omitempty"`
	WinAvServicesMatchList []string    `json:"winAvServicesMatchList,omitempty"`
}

// SyncResponse represents the response from the sync endpoint.
type SyncResponse struct {
	Data                   AgentV2Response `json:"data"`
	WinAvServicesMatchList []string        `json:"winAvServicesMatchList,omitempty"`
}

// InitDataResponse represents the response from the init endpoint.
type InitDataResponse struct {
	WinAvServicesMatchList []string `json:"winAvServicesMatchList,omitempty"`
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	StatusCode       int    `json:"statusCode"`
	Code             string `json:"code,omitempty"`
	Message          string `json:"message,omitempty"`
	SecondaryMessage string `json:"secondaryMessage,omitempty"`
}

// Client is the API client for Drata services.
type Client struct {
	httpClient *http.Client
	config     *config.Config
	dataStore  *datastore.DataStore
	version    string
}

// NewClient creates a new API client.
func NewClient(cfg *config.Config, ds *datastore.DataStore) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		config:    cfg,
		dataStore: ds,
		version:   cfg.Version,
	}
}

// doRequest performs an HTTP request with the appropriate headers.
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Determine base URL based on region in datastore or config
	region := c.dataStore.GetRegion()
	if region == "" {
		region = c.config.Region
	}
	c.config.Region = region

	url := c.config.APIHostURL() + path

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("Drata-Agent-CLI/%s (%s)", c.version, runtime.GOOS))

	uuid := c.dataStore.GetUUID()
	if uuid != "" {
		req.Header.Set("Correlation-Id", uuid)
	}

	accessToken := c.dataStore.GetAccessToken()
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	return c.httpClient.Do(req)
}

// LoginWithMagicLink authenticates using a magic link token.
func (c *Client) LoginWithMagicLink(token string) (*MeResponse, error) {
	resp, err := c.doRequest("POST", "/auth/magic-link/"+token, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleErrorResponse(resp)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, fmt.Errorf("failed to decode auth response: %w", err)
	}

	if authResp.AccessToken != "" {
		if err := c.dataStore.SetAccessToken(authResp.AccessToken); err != nil {
			return nil, fmt.Errorf("failed to save access token: %w", err)
		}
	}

	// Get user info
	return c.GetMe()
}

// GetMe retrieves the current user information.
func (c *Client) GetMe() (*MeResponse, error) {
	resp, err := c.doRequest("GET", "/users/me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var meResp MeResponse
	if err := json.NewDecoder(resp.Body).Decode(&meResp); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	// Save user to datastore
	user := &datastore.User{
		ID:                 meResp.ID,
		EntryID:            meResp.EntryID,
		Email:              meResp.Email,
		FirstName:          meResp.FirstName,
		LastName:           meResp.LastName,
		JobTitle:           meResp.JobTitle,
		AvatarURL:          meResp.AvatarURL,
		Roles:              meResp.Roles,
		DrataTermsAgreedAt: meResp.DrataTermsAgreedAt,
		CreatedAt:          meResp.CreatedAt,
		UpdatedAt:          meResp.UpdatedAt,
		Signature:          meResp.Signature,
		Language:           meResp.Language,
	}
	if err := c.dataStore.SetUser(user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return &meResp, nil
}

// Register registers the agent with the device identifiers.
func (c *Client) Register(identifiers *osquery.AgentDeviceIdentifiers) (*AgentV2Response, error) {
	resp, err := c.doRequest("POST", "/agentv2/register", identifiers)
	if err != nil {
		return nil, fmt.Errorf("failed to register: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleErrorResponse(resp)
	}

	var agentResp AgentV2Response
	if err := json.NewDecoder(resp.Body).Decode(&agentResp); err != nil {
		return nil, fmt.Errorf("failed to decode register response: %w", err)
	}

	// Save last checked at
	if agentResp.LastCheckedAt != "" {
		if err := c.dataStore.SetLastCheckedAt(agentResp.LastCheckedAt); err != nil {
			return nil, fmt.Errorf("failed to save last checked at: %w", err)
		}
	}

	return &agentResp, nil
}

// Sync sends system information to the Drata API.
func (c *Client) Sync(queryResult *osquery.QueryResult) (*SyncResponse, error) {
	resp, err := c.doRequest("POST", "/agentv2/sync", queryResult)
	if err != nil {
		return nil, fmt.Errorf("failed to sync: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.handleErrorResponse(resp)
	}

	var syncResp SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return nil, fmt.Errorf("failed to decode sync response: %w", err)
	}

	// Update datastore
	updates := map[string]interface{}{
		"complianceData": syncResp,
		"lastCheckedAt":  syncResp.Data.LastCheckedAt,
	}
	if len(syncResp.WinAvServicesMatchList) > 0 {
		updates["winAvServicesMatchList"] = syncResp.WinAvServicesMatchList
	}
	if err := c.dataStore.Update(updates); err != nil {
		return nil, fmt.Errorf("failed to update datastore: %w", err)
	}

	return &syncResp, nil
}

// GetInitData retrieves initialization data from the API.
func (c *Client) GetInitData() (*InitDataResponse, error) {
	resp, err := c.doRequest("GET", "/agentv2/init", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get init data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var initResp InitDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&initResp); err != nil {
		return nil, fmt.Errorf("failed to decode init response: %w", err)
	}

	// Save to datastore
	if len(initResp.WinAvServicesMatchList) > 0 {
		if err := c.dataStore.SetWinAvServicesMatchList(initResp.WinAvServicesMatchList); err != nil {
			return nil, fmt.Errorf("failed to save init data: %w", err)
		}
	}

	return &initResp, nil
}

// handleErrorResponse processes an error response from the API.
func (c *Client) handleErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Handle specific error codes
	switch errResp.Code {
	case "MAGIC_TOKEN_NOT_FOUND":
		return fmt.Errorf("magic token not found or expired. Please request a new registration link")
	case "REFRESH_TOKEN_NOT_FOUND":
		return fmt.Errorf("refresh token not found. Please register the agent")
	case "TOKEN_EXPIRED":
		return fmt.Errorf("authorization has expired. Please register the agent again")
	case "ACCOUNT_PENDING":
		return fmt.Errorf("account configuration is being completed. Please try again in a few minutes")
	case "ACCOUNT_MAINTENANCE":
		return fmt.Errorf("Drata is under maintenance. Please try again in a few minutes")
	case "ACCOUNT_ADMIN_DISABLED", "ACCOUNT_NON_PAYMENT":
		return fmt.Errorf("your company's account is disabled. Please contact your system administrator")
	case "ACCOUNT_USER_DELETED":
		return fmt.Errorf("your user account was deleted. Please contact your system administrator")
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: please register the agent or check your credentials")
	}

	if errResp.Message != "" {
		if errResp.SecondaryMessage != "" {
			return fmt.Errorf("%s: %s", errResp.Message, errResp.SecondaryMessage)
		}
		return fmt.Errorf("%s", errResp.Message)
	}

	return fmt.Errorf("API error (status %d)", resp.StatusCode)
}
