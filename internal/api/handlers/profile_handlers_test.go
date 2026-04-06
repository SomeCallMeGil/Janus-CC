package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"janus/internal/api/websocket"
	"janus/internal/database/sqlite"
)

// newTestServer creates an isolated httptest.Server with a fresh SQLite DB
// and the profile routes wired up. The server and DB are closed on test cleanup.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	db, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	if err := db.Connect(); err != nil {
		t.Fatalf("db.Connect: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	hub := websocket.NewHub()
	go hub.Run()

	h := New(db, hub)

	r := chi.NewRouter()
	r.Route("/api/v1/profiles", func(r chi.Router) {
		r.Get("/", h.ListProfiles)
		r.Post("/", h.CreateProfile)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetProfile)
			r.Put("/", h.UpdateProfile)
			r.Delete("/", h.DeleteProfile)
			r.Post("/generate", h.GenerateFromProfile)
		})
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// createProfile is a test helper that POSTs a profile and returns its ID.
func createProfile(t *testing.T, srv *httptest.Server, name string) string {
	t.Helper()

	body, _ := json.Marshal(map[string]interface{}{
		"name":        name,
		"description": "test profile",
		"options": map[string]interface{}{
			"output_path": filepath.Join(t.TempDir(), "payload"),
			"file_count":  10,
		},
	})

	resp, err := http.Post(srv.URL+"/api/v1/profiles", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("createProfile POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("createProfile: expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	id, _ := result["id"].(string)
	if id == "" {
		t.Fatal("createProfile: response missing id")
	}
	return id
}

func TestListProfiles(t *testing.T) {
	srv := newTestServer(t)

	resp, err := http.Get(srv.URL + "/api/v1/profiles")
	if err != nil {
		t.Fatalf("GET /profiles: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	list, ok := result["profiles"].([]interface{})
	if !ok {
		t.Fatal("response missing profiles array")
	}
	// Default profiles are seeded on startup (5 built-ins)
	if len(list) < 5 {
		t.Errorf("expected at least 5 default profiles, got %d", len(list))
	}
	if int(result["count"].(float64)) != len(list) {
		t.Errorf("count field %v does not match profiles length %d", result["count"], len(list))
	}
}

func TestCreateProfile(t *testing.T) {
	srv := newTestServer(t)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "test-create",
		"description": "integration test profile",
		"options":     map[string]interface{}{"file_count": 100},
	})

	resp, err := http.Post(srv.URL+"/api/v1/profiles", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /profiles: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var profile map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&profile)

	if id, _ := profile["id"].(string); id == "" {
		t.Error("response missing id")
	}
	if profile["name"] != "test-create" {
		t.Errorf("expected name test-create, got %v", profile["name"])
	}
	if profile["description"] != "integration test profile" {
		t.Errorf("expected description, got %v", profile["description"])
	}
}

func TestCreateProfile_MissingName(t *testing.T) {
	srv := newTestServer(t)

	body, _ := json.Marshal(map[string]interface{}{"description": "no name"})
	resp, err := http.Post(srv.URL+"/api/v1/profiles", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /profiles: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreateProfile_DuplicateName(t *testing.T) {
	srv := newTestServer(t)

	// "quick-pii-test" is seeded as a default profile
	body, _ := json.Marshal(map[string]interface{}{"name": "quick-pii-test"})
	resp, err := http.Post(srv.URL+"/api/v1/profiles", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /profiles: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for duplicate name, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["error"] == "" {
		t.Error("expected error message in response body")
	}
}

func TestGetProfile(t *testing.T) {
	srv := newTestServer(t)
	id := createProfile(t, srv, "test-get")

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/profiles/%s", srv.URL, id))
	if err != nil {
		t.Fatalf("GET /profiles/%s: %v", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var profile map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&profile)
	if profile["id"] != id {
		t.Errorf("expected id %s, got %v", id, profile["id"])
	}
	if profile["name"] != "test-get" {
		t.Errorf("expected name test-get, got %v", profile["name"])
	}
}

func TestGetProfile_NotFound(t *testing.T) {
	srv := newTestServer(t)

	resp, err := http.Get(srv.URL + "/api/v1/profiles/does-not-exist")
	if err != nil {
		t.Fatalf("GET /profiles/does-not-exist: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateProfile(t *testing.T) {
	srv := newTestServer(t)
	id := createProfile(t, srv, "test-update")

	body, _ := json.Marshal(map[string]interface{}{
		"description": "updated description",
	})
	req, _ := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/v1/profiles/%s", srv.URL, id),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT /profiles/%s: %v", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var profile map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&profile)
	if profile["description"] != "updated description" {
		t.Errorf("expected updated description, got %v", profile["description"])
	}
}

func TestDeleteProfile(t *testing.T) {
	srv := newTestServer(t)
	id := createProfile(t, srv, "test-delete")

	req, _ := http.NewRequest(http.MethodDelete,
		fmt.Sprintf("%s/api/v1/profiles/%s", srv.URL, id), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /profiles/%s: %v", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verify the profile is gone
	getResp, err := http.Get(fmt.Sprintf("%s/api/v1/profiles/%s", srv.URL, id))
	if err != nil {
		t.Fatalf("GET after delete: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", getResp.StatusCode)
	}
}

func TestGenerateFromProfile(t *testing.T) {
	srv := newTestServer(t)
	id := createProfile(t, srv, "test-generate")

	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/api/v1/profiles/%s/generate", srv.URL, id), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /profiles/%s/generate: %v", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	scenarioID, _ := result["scenario_id"].(string)
	if scenarioID == "" {
		t.Error("response missing scenario_id")
	}
	if result["profile_id"] != id {
		t.Errorf("expected profile_id %s, got %v", id, result["profile_id"])
	}
}

func TestGenerateFromProfile_NotFound(t *testing.T) {
	srv := newTestServer(t)

	req, _ := http.NewRequest(http.MethodPost,
		srv.URL+"/api/v1/profiles/does-not-exist/generate", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /profiles/does-not-exist/generate: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
