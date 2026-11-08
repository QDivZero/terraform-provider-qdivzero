package qdivzero

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInstanceLifecycle(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.EscapedPath())
		if r.Header.Get("Authorization") != "Bearer test-api-key" || r.Header.Get("X-Account-Id") != "account-1" || r.Header.Get("X-Private-Beta-Token") != "beta-1" {
			t.Errorf("missing authentication or account headers")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost, http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["name"] != "example" {
				t.Errorf("unexpected name: %#v", body)
			}
			preferences, ok := body["gpu_preferences"].([]any)
			if !ok || len(preferences) != 1 {
				t.Errorf("unexpected GPU preferences: %#v", body["gpu_preferences"])
			} else if preference, ok := preferences[0].(map[string]any); !ok || preference["public_gpu_id"] != "gpu-1" || preference["count"] != float64(2) {
				t.Errorf("unexpected GPU preference: %#v", preferences[0])
			}
			if r.Method == http.MethodPost {
				if body["huggingface_repo_id"] != "org/model" {
					t.Errorf("unexpected repository: %#v", body)
				}
				if body["workload_kind"] != "decision" {
					t.Errorf("unexpected workload kind: %#v", body)
				}
				if body["replica_count"] != float64(2) {
					t.Errorf("unexpected replica count: %#v", body)
				}
				w.WriteHeader(http.StatusCreated)
			} else if _, exists := body["huggingface_repo_id"]; exists {
				t.Error("update must not include the immutable model repository")
			}
			if r.Method == http.MethodPatch {
				if _, exists := body["workload_kind"]; exists {
					t.Error("update must not include the immutable workload kind")
				}
				if _, exists := body["replica_count"]; exists {
					t.Error("update must not include the create-only replica count")
				}
			}
			_, _ = io.WriteString(w, `{"id":"instance-1","name":"example","huggingface_repo_id":"org/model","state":"running"}`)
		case http.MethodGet:
			_, _ = io.WriteString(w, `{"id":"instance-1","name":"example","huggingface_repo_id":"org/model","state":"running"}`)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "test-api-key", AccountID: "account-1", PrivateBetaToken: "beta-1"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	preferences := []GPUPreference{{PublicGPUID: "gpu-1", Count: 2}}
	kind := "decision"
	replicas := int64(2)
	input := InstanceInput{Name: "example", HuggingfaceRepoID: "org/model", GPUPreferences: &preferences, WorkloadKind: &kind, ReplicaCount: &replicas}
	created, err := client.CreateInstance(ctx, input)
	if err != nil || created.ID != "instance-1" {
		t.Fatalf("create = %#v, %v", created, err)
	}
	if _, err := client.GetInstance(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := client.UpdateInstance(ctx, created.ID, input); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteInstance(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	want := []string{"POST /v1/instances", "GET /v1/instances/instance-1", "PATCH /v1/instances/instance-1", "DELETE /v1/instances/instance-1"}
	if strings.Join(calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestAPIErrorAndEscapedInstanceID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/v1/instances/a%2Fb" {
			t.Errorf("path = %q", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":"missing; token is test-secret and beta is beta-secret"}`)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "test-secret", PrivateBetaToken: "beta-secret"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetInstance(context.Background(), "a/b")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("error = %v, want HTTP 404", err)
	}
	if strings.Contains(err.Error(), "test-secret") || strings.Contains(err.Error(), "beta-secret") {
		t.Fatal("API error exposed credentials")
	}
}

func TestListServingEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/serving-endpoints" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"endpoints":[{"id":"ep-1","name":"model","state":"ready"}]}`)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	endpoints, err := client.ListServingEndpoints(context.Background())
	if err != nil || len(endpoints) != 1 || endpoints[0].ID != "ep-1" {
		t.Fatalf("endpoints = %#v, %v", endpoints, err)
	}
}

func TestListGPUs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/instance-config/gpus" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"gpus":[{"public_gpu_id":"gpu-1","display_name":"GPU","price_per_hour_eur":0.5,"market_type":"spot","continent":"Europe"}]}`)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	gpus, err := client.ListGPUs(context.Background())
	if err != nil || len(gpus) != 1 || gpus[0].PublicGPUID != "gpu-1" || gpus[0].PricePerHourEUR == nil || *gpus[0].PricePerHourEUR != 0.5 {
		t.Fatalf("gpus=%#v error=%v", gpus, err)
	}
}

func TestInvalidConfiguration(t *testing.T) {
	for _, cfg := range []Config{
		{},
		{APIKey: "  "},
		{APIKey: "key", BaseURL: "ftp://example.com"},
		{APIKey: "key", BaseURL: "https://user:secret@example.com"},
		{APIKey: "key", BaseURL: "https://example.com?token=secret"},
	} {
		if _, err := NewClient(cfg); err == nil {
			t.Errorf("NewClient(%#v) unexpectedly succeeded", cfg)
		}
	}
	client, err := NewClient(Config{APIKey: "key"})
	if err != nil || client.baseURL != DefaultBaseURL {
		t.Fatalf("default URL client = %#v, error = %v", client, err)
	}
}

func TestRedirectDoesNotForwardCredentials(t *testing.T) {
	var received bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	client, err := NewClient(Config{BaseURL: source.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetInstance(context.Background(), "id"); err == nil {
		t.Error("expected redirect to be returned as an API error")
	}
	if received {
		t.Fatal("followed redirect to another server")
	}
}

func TestDeleteMissingInstanceIsSuccessful(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteInstance(context.Background(), "deleted"); err != nil {
		t.Fatalf("delete already absent instance: %v", err)
	}
}

func TestEnsureInstanceStateStartsAndWaits(t *testing.T) {
	var starts, reads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/instances/i-1/start":
			starts++
			_, _ = io.WriteString(w, `{"id":"i-1","state":"pending"}`)
		case "/v1/instances/i-1":
			reads++
			state := "pending"
			if reads >= 2 {
				state = "running"
			}
			_, _ = io.WriteString(w, `{"id":"i-1","state":"`+state+`"}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	initialState := "stopped"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := client.EnsureInstanceState(ctx, "i-1", Instance{ID: "i-1", State: &initialState}, "running", time.Millisecond)
	if err != nil || result.State == nil || *result.State != "running" || starts != 1 || reads < 2 {
		t.Fatalf("result=%#v, starts=%d, reads=%d, error=%v", result, starts, reads, err)
	}
}

func TestEnsureInstanceStateFailsOnTerminalFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"id":"i-1","state":"failed"}`)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	stopped := "stopped"
	_, err = client.EnsureInstanceState(context.Background(), "i-1", Instance{ID: "i-1", State: &stopped}, "running", time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "failed") {
		t.Fatalf("error = %v, want terminal failure", err)
	}
}

func TestEnsureRunningRetriesFailedInstanceOnce(t *testing.T) {
	var starts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		starts++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/instances/i-1/start" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"id":"i-1","state":"running"}`)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	failed := "failed"
	result, err := client.EnsureInstanceState(context.Background(), "i-1", Instance{ID: "i-1", State: &failed}, "running", time.Millisecond)
	if err != nil || result.State == nil || *result.State != "running" || starts != 1 {
		t.Fatalf("result=%#v starts=%d error=%v", result, starts, err)
	}
}

func TestEnsureInstanceStateStopsPendingAndTreatsNonBootstrapAsStopped(t *testing.T) {
	var stops int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/instances/i-1/stop" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		stops++
		_, _ = io.WriteString(w, `{"id":"i-1","state":"stopped"}`)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	pending := "pending"
	if _, err := client.EnsureInstanceState(context.Background(), "i-1", Instance{ID: "i-1", State: &pending}, "stopped", time.Millisecond); err != nil {
		t.Fatal(err)
	}
	nonBootstrap := "non_bootstrap"
	if _, err := client.EnsureInstanceState(context.Background(), "i-1", Instance{ID: "i-1", State: &nonBootstrap}, "stopped", time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if stops != 1 {
		t.Fatalf("stop calls = %d, want 1", stops)
	}
}

func TestEnsureInstanceStateHonorsContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"id":"i-1","state":"pending"}`)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	pending := "pending"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = client.EnsureInstanceState(ctx, "i-1", Instance{ID: "i-1", State: &pending}, "running", 5*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v, want context deadline exceeded", err)
	}
}
