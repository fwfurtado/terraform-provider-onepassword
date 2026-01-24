package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/1password/onepassword-sdk-go"
	"github.com/stretchr/testify/require"
)

const mockoonSampleURL = "https://raw.githubusercontent.com/mockoon/mock-samples/main/mock-apis/data/1passwordlocal-connect.json"

type mockoonEnvironment struct {
	Routes []mockoonRoute `json:"routes"`
}

type mockoonRoute struct {
	Method    string            `json:"method"`
	Endpoint  string            `json:"endpoint"`
	Responses []mockoonResponse `json:"responses"`
}

type mockoonResponse struct {
	StatusCode int             `json:"statusCode"`
	Body       string          `json:"body"`
	Headers    []mockoonHeader `json:"headers"`
	Default    bool            `json:"default"`
}

type mockoonHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type mockoonServer struct {
	server  *httptest.Server
	store   *mockoonStore
	routes  []mockoonRoute
	fixture mockoonFixture
}

func newMockoonServer(t *testing.T) *mockoonServer {
	t.Helper()

	env := loadMockoonEnvironment(t)
	store, fixture := newMockoonStore()

	srv := &mockoonServer{
		store:  store,
		routes: env.Routes,
	}
	srv.server = httptest.NewServer(http.HandlerFunc(srv.handle))
	srv.fixture = fixture

	return srv
}

func (s *mockoonServer) Close() {
	s.server.Close()
}

func (s *mockoonServer) Fixture() mockoonFixture {
	return s.fixture
}

func (s *mockoonServer) NewClient() opClient {
	return newMockoonClient(s.server.URL, s.store)
}

func loadMockoonEnvironment(t *testing.T) mockoonEnvironment {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mockoonSampleURL, nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var env mockoonEnvironment
	require.NoError(t, json.Unmarshal(body, &env))

	requireMockoonRoutes(t, env.Routes)

	return env
}

func requireMockoonRoutes(t *testing.T, routes []mockoonRoute) {
	t.Helper()

	required := []struct {
		method   string
		endpoint string
	}{
		{method: "get", endpoint: "vaults"},
		{method: "get", endpoint: "vaults/:vaultUuid/items"},
		{method: "post", endpoint: "vaults/:vaultUuid/items"},
		{method: "get", endpoint: "vaults/:vaultUuid/items/:itemUuid"},
		{method: "put", endpoint: "vaults/:vaultUuid/items/:itemUuid"},
		{method: "delete", endpoint: "vaults/:vaultUuid/items/:itemUuid"},
	}

	for _, entry := range required {
		found := false
		for _, route := range routes {
			if strings.EqualFold(route.Method, entry.method) && route.Endpoint == entry.endpoint {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("mockoon sample missing route %s %s", entry.method, entry.endpoint)
		}
	}
}

func (s *mockoonServer) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	route, params, ok := matchMockoonRoute(s.routes, r.Method, path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch strings.ToLower(r.Method) + " " + route.Endpoint {
	case "get vaults":
		s.handleVaults(w)
		return
	case "get vaults/:vaultUuid/items":
		s.handleItemsList(w, params)
		return
	case "post vaults/:vaultUuid/items":
		s.handleItemCreate(w, r, params)
		return
	case "get vaults/:vaultUuid/items/:itemUuid":
		s.handleItemGet(w, r, params)
		return
	case "put vaults/:vaultUuid/items/:itemUuid", "patch vaults/:vaultUuid/items/:itemUuid":
		s.handleItemUpdate(w, r, params)
		return
	case "delete vaults/:vaultUuid/items/:itemUuid":
		s.handleItemDelete(w, r, params)
		return
	default:
		s.writeMockoonResponse(w, route)
		return
	}
}

func (s *mockoonServer) handleVaults(w http.ResponseWriter) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	vaults := make([]connectVault, 0, len(s.store.vaults))
	for _, vault := range s.store.vaults {
		vaults = append(vaults, *vault)
	}

	writeJSON(w, http.StatusOK, vaults)
}

func (s *mockoonServer) handleItemsList(w http.ResponseWriter, params map[string]string) {
	vaultID := params["vaultUuid"]

	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	items := make([]connectItemOverview, 0)
	for _, item := range s.store.items[vaultID] {
		items = append(items, connectItemToOverview(*item))
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *mockoonServer) handleItemGet(w http.ResponseWriter, r *http.Request, params map[string]string) {
	vaultID := params["vaultUuid"]
	itemID := params["itemUuid"]

	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	item, ok := s.store.items[vaultID][itemID]
	if !ok {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (s *mockoonServer) handleItemCreate(w http.ResponseWriter, r *http.Request, params map[string]string) {
	var req connectItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	vaultID := params["vaultUuid"]
	now := time.Now().UTC().Format(time.RFC3339)

	item := connectItem{
		ID:        s.store.nextItemID(),
		Title:     req.Title,
		Category:  req.Category,
		Vault:     connectVaultRef{ID: vaultID},
		Fields:    req.Fields,
		Sections:  req.Sections,
		Tags:      req.Tags,
		URLs:      req.URLs,
		Notes:     req.Notes,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	if s.store.items[vaultID] == nil {
		s.store.items[vaultID] = map[string]*connectItem{}
	}
	s.store.items[vaultID][item.ID] = &item

	writeJSON(w, http.StatusOK, item)
}

func (s *mockoonServer) handleItemUpdate(w http.ResponseWriter, r *http.Request, params map[string]string) {
	vaultID := params["vaultUuid"]
	itemID := params["itemUuid"]

	var req connectItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	item, ok := s.store.items[vaultID][itemID]
	if !ok {
		http.NotFound(w, r)
		return
	}

	item.Title = req.Title
	item.Category = req.Category
	item.Fields = req.Fields
	item.Sections = req.Sections
	item.Tags = req.Tags
	item.URLs = req.URLs
	item.Notes = req.Notes
	item.Version++
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	writeJSON(w, http.StatusOK, item)
}

func (s *mockoonServer) handleItemDelete(w http.ResponseWriter, r *http.Request, params map[string]string) {
	vaultID := params["vaultUuid"]
	itemID := params["itemUuid"]

	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	if _, ok := s.store.items[vaultID][itemID]; !ok {
		http.NotFound(w, r)
		return
	}

	delete(s.store.items[vaultID], itemID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *mockoonServer) writeMockoonResponse(w http.ResponseWriter, route mockoonRoute) {
	response := selectMockoonResponse(route)
	for _, header := range response.Headers {
		w.Header().Set(header.Key, header.Value)
	}
	w.WriteHeader(response.StatusCode)
	if response.Body == "" {
		return
	}
	_, _ = io.WriteString(w, renderMockoonTemplate(response.Body))
}

func selectMockoonResponse(route mockoonRoute) mockoonResponse {
	for _, response := range route.Responses {
		if response.Default {
			return response
		}
	}
	if len(route.Responses) > 0 {
		return route.Responses[0]
	}
	return mockoonResponse{StatusCode: http.StatusNotFound}
}

func renderMockoonTemplate(body string) string {
	replacements := map[string]string{
		"{{faker 'datatype.boolean'}}":           "true",
		"{{faker 'date.recent' 365}}":            "2024-01-01T00:00:00Z",
		"{{faker 'number.int' max=99999}}":       "12345",
		"{{faker 'string.uuid'}}":                "11111111-1111-1111-1111-111111111111",
		"{{oneOf (array 'ARCHIVED' 'DELETED')}}": "ARCHIVED",
		"{{oneOf (array 'ITEM' 'VAULT')}}":       "ITEM",
		"{{oneOf (array 'LOGIN' 'PASSWORD' 'API_CREDENTIAL' 'SERVER' 'DATABASE' 'CREDIT_CARD' 'MEMBERSHIP' 'PASSPORT' 'SOFTWARE_LICENSE' 'OUTDOOR_LICENSE' 'SECURE_NOTE' 'WIRELESS_ROUTER' 'BANK_ACCOUNT' 'DRIVER_LICENSE' 'IDENTITY' 'REWARD_PROGRAM' 'DOCUMENT' 'EMAIL_ACCOUNT' 'SOCIAL_SECURITY_NUMBER' 'CUSTOM')}}": "LOGIN",
		"{{oneOf (array 'READ' 'CREATE' 'UPDATE' 'DELETE')}}":               "READ",
		"{{oneOf (array 'SUCCESS' 'DENY')}}":                                "SUCCESS",
		"{{oneOf (array 'USER_CREATED' 'PERSONAL' 'EVERYONE' 'TRANSFER')}}": "USER_CREATED",
	}

	for needle, replacement := range replacements {
		body = strings.ReplaceAll(body, needle, replacement)
	}
	return body
}

func matchMockoonRoute(routes []mockoonRoute, method, path string) (mockoonRoute, map[string]string, bool) {
	for _, route := range routes {
		if !strings.EqualFold(route.Method, method) {
			continue
		}
		if params, ok := matchMockoonEndpoint(route.Endpoint, path); ok {
			return route, params, true
		}
	}
	return mockoonRoute{}, nil, false
}

func matchMockoonEndpoint(pattern, path string) (map[string]string, bool) {
	pattern = strings.Trim(pattern, "/")
	path = strings.Trim(path, "/")

	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	if len(patternParts) != len(pathParts) {
		return nil, false
	}

	params := map[string]string{}
	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			params[strings.TrimPrefix(part, ":")] = pathParts[i]
			continue
		}
		if part != pathParts[i] {
			return nil, false
		}
	}
	return params, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type mockoonStore struct {
	mu     sync.RWMutex
	vaults map[string]*connectVault
	items  map[string]map[string]*connectItem
	nextID int
}

func newMockoonStore() (*mockoonStore, mockoonFixture) {
	now := time.Now().UTC().Format(time.RFC3339)
	vaultID := "vault-001"
	vaultName := "Mockoon Vault"
	itemID := "item-001"
	itemTitle := "Mockoon Item"
	sectionName := "credentials"

	vault := &connectVault{
		ID:               vaultID,
		Name:             vaultName,
		Type:             "USER_CREATED",
		Description:      "Mockoon fixture vault",
		Items:            1,
		AttributeVersion: 1,
		ContentVersion:   1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	usernameField := connectField{
		ID:    "username",
		Label: "username",
		Type:  "Text",
		Value: "mock-user",
	}
	passwordField := connectField{
		ID:      "password",
		Label:   "password",
		Type:    "Concealed",
		Value:   "mock-password",
		Section: sectionName,
	}
	item := &connectItem{
		ID:       itemID,
		Title:    itemTitle,
		Category: "LOGIN",
		Vault: connectVaultRef{
			ID: vaultID,
		},
		Fields: []connectField{
			usernameField,
			passwordField,
		},
		Sections: []connectSection{
			{ID: sectionName, Label: sectionName},
		},
		Tags: []string{"mock"},
		URLs: []connectURL{
			{Href: "https://example.com", Primary: true},
			{Href: "https://example.org"},
		},
		Notes:     "seed item",
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	store := &mockoonStore{
		vaults: map[string]*connectVault{
			vaultID: vault,
		},
		items: map[string]map[string]*connectItem{
			vaultID: {
				itemID: item,
			},
		},
		nextID: 2,
	}

	return store, mockoonFixture{
		VaultID:       vaultID,
		VaultName:     vaultName,
		ItemID:        itemID,
		ItemTitle:     itemTitle,
		SectionName:   sectionName,
		UsernameField: usernameField.Label,
		UsernameValue: usernameField.Value,
		PasswordField: passwordField.Label,
		PasswordValue: passwordField.Value,
	}
}

func (s *mockoonStore) nextItemID() string {
	id := fmt.Sprintf("item-%03d", s.nextID)
	s.nextID++
	return id
}

type connectVault struct {
	AttributeVersion int    `json:"attributeVersion"`
	ContentVersion   int    `json:"contentVersion"`
	CreatedAt        string `json:"createdAt"`
	Description      string `json:"description"`
	ID               string `json:"id"`
	Items            int    `json:"items"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	UpdatedAt        string `json:"updatedAt"`
}

type connectVaultRef struct {
	ID string `json:"id"`
}

type connectURL struct {
	Href    string `json:"href"`
	Primary bool   `json:"primary,omitempty"`
}

type connectField struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Section string `json:"section,omitempty"`
}

type connectSection struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type connectItem struct {
	ID        string           `json:"id"`
	Title     string           `json:"title"`
	Category  string           `json:"category"`
	Vault     connectVaultRef  `json:"vault"`
	Fields    []connectField   `json:"fields"`
	Sections  []connectSection `json:"sections"`
	Tags      []string         `json:"tags"`
	URLs      []connectURL     `json:"urls"`
	Notes     string           `json:"notes"`
	Version   int              `json:"version"`
	CreatedAt string           `json:"createdAt"`
	UpdatedAt string           `json:"updatedAt"`
}

type connectItemOverview struct {
	Category  string          `json:"category"`
	CreatedAt string          `json:"createdAt"`
	ID        string          `json:"id"`
	State     string          `json:"state"`
	Tags      []string        `json:"tags"`
	Title     string          `json:"title"`
	UpdatedAt string          `json:"updatedAt"`
	URLs      []connectURL    `json:"urls"`
	Vault     connectVaultRef `json:"vault"`
	Version   int             `json:"version"`
}

type connectItemRequest struct {
	Title    string           `json:"title"`
	Category string           `json:"category"`
	Vault    connectVaultRef  `json:"vault"`
	Fields   []connectField   `json:"fields,omitempty"`
	Sections []connectSection `json:"sections,omitempty"`
	Tags     []string         `json:"tags,omitempty"`
	URLs     []connectURL     `json:"urls,omitempty"`
	Notes    string           `json:"notes,omitempty"`
}

func connectItemToOverview(item connectItem) connectItemOverview {
	return connectItemOverview{
		Category:  item.Category,
		CreatedAt: item.CreatedAt,
		ID:        item.ID,
		State:     "ARCHIVED",
		Tags:      item.Tags,
		Title:     item.Title,
		UpdatedAt: item.UpdatedAt,
		URLs:      item.URLs,
		Vault:     item.Vault,
		Version:   item.Version,
	}
}

type mockoonClient struct {
	baseURL    string
	httpClient *http.Client
	store      *mockoonStore
}

func newMockoonClient(baseURL string, store *mockoonStore) *mockoonClient {
	return &mockoonClient{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{Timeout: 5 * time.Second},
		store:      store,
	}
}

func (c *mockoonClient) Secrets() onepassword.SecretsAPI {
	return &mockoonSecretsAPI{store: c.store}
}

func (c *mockoonClient) Items() onepassword.ItemsAPI {
	return &mockoonItemsAPI{baseURL: c.baseURL, httpClient: c.httpClient}
}

func (c *mockoonClient) Vaults() onepassword.VaultsAPI {
	return &mockoonVaultsAPI{baseURL: c.baseURL, httpClient: c.httpClient}
}

type mockoonSecretsAPI struct {
	store *mockoonStore
}

func (s *mockoonSecretsAPI) Resolve(ctx context.Context, secretReference string) (string, error) {
	parsed, err := parseSecretReference(secretReference)
	if err != nil {
		return "", err
	}

	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	vault, ok := s.findVaultByName(parsed.Vault)
	if !ok {
		return "", newOnePasswordError("vault not found", OnePasswordErrorNotFound)
	}

	item := s.findItemByTitle(vault.ID, parsed.Item)
	if item == nil {
		return "", newOnePasswordError("item not found", OnePasswordErrorNotFound)
	}

	if parsed.Section != "" && !s.sectionExists(item, parsed.Section) {
		return "", ErrSectionNotFound(parsed.Item, parsed.Section)
	}

	for _, field := range item.Fields {
		if field.Label != parsed.Field {
			continue
		}
		if parsed.Section != "" && !strings.EqualFold(field.Section, parsed.Section) {
			continue
		}
		return field.Value, nil
	}

	return "", ErrFieldNotFound(parsed.Item, parsed.Field, parsed.Section)
}

func (s *mockoonSecretsAPI) ResolveAll(ctx context.Context, secretReferences []string) (onepassword.ResolveAllResponse, error) {
	responses := make(map[string]onepassword.Response[onepassword.ResolvedReference, onepassword.ResolveReferenceError])
	for _, ref := range secretReferences {
		secret, err := s.Resolve(ctx, ref)
		if err != nil {
			responses[ref] = onepassword.Response[onepassword.ResolvedReference, onepassword.ResolveReferenceError]{
				Error: &onepassword.ResolveReferenceError{Type: onepassword.ResolveReferenceErrorTypeVariantOther},
			}
			continue
		}
		responses[ref] = onepassword.Response[onepassword.ResolvedReference, onepassword.ResolveReferenceError]{
			Content: &onepassword.ResolvedReference{Secret: secret},
		}
	}
	return onepassword.ResolveAllResponse{IndividualResponses: responses}, nil
}

func (s *mockoonSecretsAPI) findVaultByName(name string) (*connectVault, bool) {
	for _, vault := range s.store.vaults {
		if strings.EqualFold(vault.Name, name) {
			return vault, true
		}
	}
	return nil, false
}

func (s *mockoonSecretsAPI) findItemByTitle(vaultID, title string) *connectItem {
	for _, item := range s.store.items[vaultID] {
		if strings.EqualFold(item.Title, title) {
			return item
		}
	}
	return nil
}

func (s *mockoonSecretsAPI) sectionExists(item *connectItem, section string) bool {
	for _, candidate := range item.Sections {
		if strings.EqualFold(candidate.Label, section) || strings.EqualFold(candidate.ID, section) {
			return true
		}
	}
	return false
}

type mockoonItemsAPI struct {
	baseURL    string
	httpClient *http.Client
}

func (m *mockoonItemsAPI) Create(ctx context.Context, params onepassword.ItemCreateParams) (onepassword.Item, error) {
	req := connectItemRequest{
		Title:    params.Title,
		Category: toConnectCategory(params.Category),
		Vault:    connectVaultRef{ID: params.VaultID},
		Fields:   toConnectFields(params.Fields),
		Sections: toConnectSections(params.Sections),
		Tags:     params.Tags,
		URLs:     toConnectURLs(params.Websites),
	}
	if params.Notes != nil {
		req.Notes = *params.Notes
	}

	var response connectItem
	if err := m.doRequest(ctx, http.MethodPost, fmt.Sprintf("/vaults/%s/items", params.VaultID), req, &response); err != nil {
		return onepassword.Item{}, err
	}
	return connectItemToSDK(response), nil
}

func (m *mockoonItemsAPI) CreateAll(ctx context.Context, vaultID string, params []onepassword.ItemCreateParams) (onepassword.ItemsUpdateAllResponse, error) {
	return onepassword.ItemsUpdateAllResponse{}, errors.New("not implemented")
}

func (m *mockoonItemsAPI) Get(ctx context.Context, vaultID string, itemID string) (onepassword.Item, error) {
	var response connectItem
	if err := m.doRequest(ctx, http.MethodGet, fmt.Sprintf("/vaults/%s/items/%s", vaultID, itemID), nil, &response); err != nil {
		return onepassword.Item{}, err
	}
	return connectItemToSDK(response), nil
}

func (m *mockoonItemsAPI) GetAll(ctx context.Context, vaultID string, itemIds []string) (onepassword.ItemsGetAllResponse, error) {
	return onepassword.ItemsGetAllResponse{}, errors.New("not implemented")
}

func (m *mockoonItemsAPI) Put(ctx context.Context, item onepassword.Item) (onepassword.Item, error) {
	req := connectItemRequest{
		Title:    item.Title,
		Category: toConnectCategory(item.Category),
		Vault:    connectVaultRef{ID: item.VaultID},
		Fields:   toConnectFields(item.Fields),
		Sections: toConnectSections(item.Sections),
		Tags:     item.Tags,
		URLs:     toConnectURLs(item.Websites),
		Notes:    item.Notes,
	}

	var response connectItem
	if err := m.doRequest(ctx, http.MethodPut, fmt.Sprintf("/vaults/%s/items/%s", item.VaultID, item.ID), req, &response); err != nil {
		return onepassword.Item{}, err
	}
	return connectItemToSDK(response), nil
}

func (m *mockoonItemsAPI) Delete(ctx context.Context, vaultID string, itemID string) error {
	return m.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/vaults/%s/items/%s", vaultID, itemID), nil, nil)
}

func (m *mockoonItemsAPI) DeleteAll(ctx context.Context, vaultID string, itemIds []string) (onepassword.ItemsDeleteAllResponse, error) {
	return onepassword.ItemsDeleteAllResponse{}, errors.New("not implemented")
}

func (m *mockoonItemsAPI) Archive(ctx context.Context, vaultID string, itemID string) error {
	return errors.New("not implemented")
}

func (m *mockoonItemsAPI) List(ctx context.Context, vaultID string, filters ...onepassword.ItemListFilter) ([]onepassword.ItemOverview, error) {
	var response []connectItemOverview
	if err := m.doRequest(ctx, http.MethodGet, fmt.Sprintf("/vaults/%s/items", vaultID), nil, &response); err != nil {
		return nil, err
	}
	overviews := make([]onepassword.ItemOverview, 0, len(response))
	for _, item := range response {
		overviews = append(overviews, connectOverviewToSDK(item))
	}
	return overviews, nil
}

func (m *mockoonItemsAPI) Shares() onepassword.ItemsSharesAPI {
	return mockoonItemsSharesAPI{}
}

func (m *mockoonItemsAPI) Files() onepassword.ItemsFilesAPI {
	return mockoonItemsFilesAPI{}
}

func (m *mockoonItemsAPI) doRequest(ctx context.Context, method, path string, payload any, target any) (err error) {
	fullURL, err := url.JoinPath(m.baseURL, path)
	if err != nil {
		return err
	}

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("mockoon request failed with status %d", resp.StatusCode)
	}

	if target == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

type mockoonItemsSharesAPI struct{}

func (m mockoonItemsSharesAPI) GetAccountPolicy(ctx context.Context, vaultID string, itemID string) (onepassword.ItemShareAccountPolicy, error) {
	return onepassword.ItemShareAccountPolicy{}, errors.New("not implemented")
}

func (m mockoonItemsSharesAPI) ValidateRecipients(ctx context.Context, policy onepassword.ItemShareAccountPolicy, recipients []string) ([]onepassword.ValidRecipient, error) {
	return nil, errors.New("not implemented")
}

func (m mockoonItemsSharesAPI) Create(ctx context.Context, item onepassword.Item, policy onepassword.ItemShareAccountPolicy, params onepassword.ItemShareParams) (string, error) {
	return "", errors.New("not implemented")
}

type mockoonItemsFilesAPI struct{}

func (m mockoonItemsFilesAPI) Attach(ctx context.Context, item onepassword.Item, fileParams onepassword.FileCreateParams) (onepassword.Item, error) {
	return onepassword.Item{}, errors.New("not implemented")
}

func (m mockoonItemsFilesAPI) Read(ctx context.Context, vaultID string, itemID string, attr onepassword.FileAttributes) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (m mockoonItemsFilesAPI) Delete(ctx context.Context, item onepassword.Item, sectionID string, fieldID string) (onepassword.Item, error) {
	return onepassword.Item{}, errors.New("not implemented")
}

func (m mockoonItemsFilesAPI) ReplaceDocument(ctx context.Context, item onepassword.Item, docParams onepassword.DocumentCreateParams) (onepassword.Item, error) {
	return onepassword.Item{}, errors.New("not implemented")
}

type mockoonVaultsAPI struct {
	baseURL    string
	httpClient *http.Client
}

func (m *mockoonVaultsAPI) List(ctx context.Context, params ...onepassword.VaultListParams) ([]onepassword.VaultOverview, error) {
	var response []connectVault
	if err := m.doRequest(ctx, http.MethodGet, "/vaults", nil, &response); err != nil {
		return nil, err
	}

	vaults := make([]onepassword.VaultOverview, 0, len(response))
	for _, vault := range response {
		vaults = append(vaults, connectVaultToSDK(vault))
	}
	return vaults, nil
}

func (m *mockoonVaultsAPI) GetOverview(ctx context.Context, vaultUuid string) (onepassword.VaultOverview, error) {
	return onepassword.VaultOverview{}, errors.New("not implemented")
}

func (m *mockoonVaultsAPI) Get(ctx context.Context, vaultUuid string, vaultParams onepassword.VaultGetParams) (onepassword.Vault, error) {
	return onepassword.Vault{}, errors.New("not implemented")
}

func (m *mockoonVaultsAPI) GrantGroupPermissions(ctx context.Context, vaultID string, groupPermissionsList []onepassword.GroupAccess) error {
	return errors.New("not implemented")
}

func (m *mockoonVaultsAPI) UpdateGroupPermissions(ctx context.Context, groupPermissionsList []onepassword.GroupVaultAccess) error {
	return errors.New("not implemented")
}

func (m *mockoonVaultsAPI) RevokeGroupPermissions(ctx context.Context, vaultID string, groupID string) error {
	return errors.New("not implemented")
}

func (m *mockoonVaultsAPI) doRequest(ctx context.Context, method, path string, payload any, target any) (err error) {
	fullURL, err := url.JoinPath(m.baseURL, path)
	if err != nil {
		return err
	}

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("mockoon request failed with status %d", resp.StatusCode)
	}

	if target == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func connectVaultToSDK(vault connectVault) onepassword.VaultOverview {
	return onepassword.VaultOverview{
		ID:               vault.ID,
		Title:            vault.Name,
		Description:      vault.Description,
		VaultType:        connectVaultType(vault.Type),
		ActiveItemCount:  uint32(vault.Items),
		ContentVersion:   uint32(vault.ContentVersion),
		AttributeVersion: uint32(vault.AttributeVersion),
		CreatedAt:        parseConnectTime(vault.CreatedAt),
		UpdatedAt:        parseConnectTime(vault.UpdatedAt),
	}
}

func connectOverviewToSDK(item connectItemOverview) onepassword.ItemOverview {
	return onepassword.ItemOverview{
		ID:        item.ID,
		Title:     item.Title,
		Category:  connectCategoryToSDK(item.Category),
		VaultID:   item.Vault.ID,
		Websites:  connectURLsToWebsites(item.URLs),
		Tags:      item.Tags,
		CreatedAt: parseConnectTime(item.CreatedAt),
		UpdatedAt: parseConnectTime(item.UpdatedAt),
		State:     onepassword.ItemStateActive,
	}
}

func connectItemToSDK(item connectItem) onepassword.Item {
	fields := make([]onepassword.ItemField, 0, len(item.Fields))
	for _, field := range item.Fields {
		converted := onepassword.ItemField{
			ID:        field.ID,
			Title:     field.Label,
			FieldType: connectFieldTypeToSDK(field.Type),
			Value:     field.Value,
		}
		if field.Section != "" {
			sectionID := field.Section
			converted.SectionID = &sectionID
		}
		fields = append(fields, converted)
	}

	sections := make([]onepassword.ItemSection, 0, len(item.Sections))
	for _, section := range item.Sections {
		sections = append(sections, onepassword.ItemSection{
			ID:    section.ID,
			Title: section.Label,
		})
	}

	return onepassword.Item{
		ID:        item.ID,
		Title:     item.Title,
		Category:  connectCategoryToSDK(item.Category),
		VaultID:   item.Vault.ID,
		Fields:    fields,
		Sections:  sections,
		Notes:     item.Notes,
		Tags:      item.Tags,
		Websites:  connectURLsToWebsites(item.URLs),
		Version:   uint32(item.Version),
		CreatedAt: parseConnectTime(item.CreatedAt),
		UpdatedAt: parseConnectTime(item.UpdatedAt),
	}
}

func connectURLsToWebsites(urls []connectURL) []onepassword.Website {
	websites := make([]onepassword.Website, 0, len(urls))
	for _, candidate := range urls {
		websites = append(websites, onepassword.Website{
			URL:              candidate.Href,
			Label:            "website",
			AutofillBehavior: onepassword.AutofillBehaviorAnywhereOnWebsite,
		})
	}
	return websites
}

func toConnectFields(fields []onepassword.ItemField) []connectField {
	if len(fields) == 0 {
		return nil
	}
	out := make([]connectField, 0, len(fields))
	for _, field := range fields {
		item := connectField{
			ID:    field.ID,
			Label: field.Title,
			Type:  string(field.FieldType),
			Value: field.Value,
		}
		if field.SectionID != nil {
			item.Section = *field.SectionID
		}
		out = append(out, item)
	}
	return out
}

func toConnectSections(sections []onepassword.ItemSection) []connectSection {
	if len(sections) == 0 {
		return nil
	}
	out := make([]connectSection, 0, len(sections))
	for _, section := range sections {
		out = append(out, connectSection{
			ID:    section.ID,
			Label: section.Title,
		})
	}
	return out
}

func toConnectURLs(websites []onepassword.Website) []connectURL {
	if len(websites) == 0 {
		return nil
	}
	out := make([]connectURL, 0, len(websites))
	for i, site := range websites {
		out = append(out, connectURL{
			Href:    site.URL,
			Primary: i == 0,
		})
	}
	return out
}

func toConnectCategory(category onepassword.ItemCategory) string {
	switch category {
	case onepassword.ItemCategoryLogin:
		return "LOGIN"
	case onepassword.ItemCategorySecureNote:
		return "SECURE_NOTE"
	case onepassword.ItemCategoryPassword:
		return "PASSWORD"
	case onepassword.ItemCategoryAPICredentials:
		return "API_CREDENTIAL"
	case onepassword.ItemCategoryServer:
		return "SERVER"
	case onepassword.ItemCategoryDatabase:
		return "DATABASE"
	case onepassword.ItemCategoryCreditCard:
		return "CREDIT_CARD"
	case onepassword.ItemCategoryMembership:
		return "MEMBERSHIP"
	case onepassword.ItemCategoryPassport:
		return "PASSPORT"
	case onepassword.ItemCategorySoftwareLicense:
		return "SOFTWARE_LICENSE"
	case onepassword.ItemCategoryOutdoorLicense:
		return "OUTDOOR_LICENSE"
	case onepassword.ItemCategoryRouter:
		return "WIRELESS_ROUTER"
	case onepassword.ItemCategoryBankAccount:
		return "BANK_ACCOUNT"
	case onepassword.ItemCategoryDriverLicense:
		return "DRIVER_LICENSE"
	case onepassword.ItemCategoryIdentity:
		return "IDENTITY"
	case onepassword.ItemCategoryRewards:
		return "REWARD_PROGRAM"
	case onepassword.ItemCategoryDocument:
		return "DOCUMENT"
	case onepassword.ItemCategoryEmail:
		return "EMAIL_ACCOUNT"
	case onepassword.ItemCategorySocialSecurityNumber:
		return "SOCIAL_SECURITY_NUMBER"
	default:
		return "CUSTOM"
	}
}

func connectCategoryToSDK(category string) onepassword.ItemCategory {
	switch strings.ToUpper(category) {
	case "LOGIN":
		return onepassword.ItemCategoryLogin
	case "SECURE_NOTE":
		return onepassword.ItemCategorySecureNote
	case "PASSWORD":
		return onepassword.ItemCategoryPassword
	case "API_CREDENTIAL":
		return onepassword.ItemCategoryAPICredentials
	case "SERVER":
		return onepassword.ItemCategoryServer
	case "DATABASE":
		return onepassword.ItemCategoryDatabase
	case "CREDIT_CARD":
		return onepassword.ItemCategoryCreditCard
	case "MEMBERSHIP":
		return onepassword.ItemCategoryMembership
	case "PASSPORT":
		return onepassword.ItemCategoryPassport
	case "SOFTWARE_LICENSE":
		return onepassword.ItemCategorySoftwareLicense
	case "OUTDOOR_LICENSE":
		return onepassword.ItemCategoryOutdoorLicense
	case "WIRELESS_ROUTER":
		return onepassword.ItemCategoryRouter
	case "BANK_ACCOUNT":
		return onepassword.ItemCategoryBankAccount
	case "DRIVER_LICENSE":
		return onepassword.ItemCategoryDriverLicense
	case "IDENTITY":
		return onepassword.ItemCategoryIdentity
	case "REWARD_PROGRAM":
		return onepassword.ItemCategoryRewards
	case "DOCUMENT":
		return onepassword.ItemCategoryDocument
	case "EMAIL_ACCOUNT":
		return onepassword.ItemCategoryEmail
	case "SOCIAL_SECURITY_NUMBER":
		return onepassword.ItemCategorySocialSecurityNumber
	default:
		return onepassword.ItemCategoryUnsupported
	}
}

func connectVaultType(vaultType string) onepassword.VaultType {
	switch strings.ToUpper(vaultType) {
	case "USER_CREATED":
		return onepassword.VaultTypeUserCreated
	case "PERSONAL":
		return onepassword.VaultTypePersonal
	case "EVERYONE":
		return onepassword.VaultTypeEveryone
	case "TRANSFER":
		return onepassword.VaultTypeTransfer
	default:
		return onepassword.VaultTypeUnsupported
	}
}

func connectFieldTypeToSDK(fieldType string) onepassword.ItemFieldType {
	switch strings.ToLower(fieldType) {
	case "concealed", "password":
		return onepassword.ItemFieldTypeConcealed
	case "email":
		return onepassword.ItemFieldTypeEmail
	case "url", "website":
		return onepassword.ItemFieldTypeURL
	case "totp", "otp":
		return onepassword.ItemFieldTypeTOTP
	default:
		return onepassword.ItemFieldTypeText
	}
}

func parseConnectTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed
	}
	return time.Now().UTC()
}

type secretReference struct {
	Vault   string
	Item    string
	Section string
	Field   string
}

func parseSecretReference(reference string) (secretReference, error) {
	if !strings.HasPrefix(reference, "op://") {
		return secretReference{}, fmt.Errorf("invalid secret reference")
	}
	parts := strings.Split(strings.TrimPrefix(reference, "op://"), "/")
	if len(parts) < 3 {
		return secretReference{}, fmt.Errorf("invalid secret reference")
	}
	if len(parts) == 3 {
		return secretReference{
			Vault: parts[0],
			Item:  parts[1],
			Field: parts[2],
		}, nil
	}
	return secretReference{
		Vault:   parts[0],
		Item:    parts[1],
		Section: parts[2],
		Field:   parts[3],
	}, nil
}
