package qdivzero

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVectorDatabaseLifecycle(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost, http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["embedding_serving_name"] != "embeddings" || body["name"] != "docs" {
				t.Errorf("body=%#v", body)
			}
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			}
			_, _ = io.WriteString(w, `{"id":"db-1","name":"docs","slug":"docs","embedding_serving_name":"embeddings","status":"ready"}`)
		case http.MethodGet:
			_, _ = io.WriteString(w, `{"id":"db-1","name":"docs","embedding_serving_name":"embeddings","status":"ready"}`)
		case http.MethodDelete:
			_, _ = io.WriteString(w, `{"deleted":true}`)
		}
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	input := VectorDatabaseInput{Name: "docs", EmbeddingServingName: "embeddings"}
	created, err := client.CreateVectorDatabase(context.Background(), input)
	if err != nil || created.ID != "db-1" {
		t.Fatalf("create=%#v error=%v", created, err)
	}
	if _, err := client.GetVectorDatabase(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := client.UpdateVectorDatabase(context.Background(), created.ID, input); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteVectorDatabase(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	want := "POST /v1/flexible-vector-databases,GET /v1/flexible-vector-databases/db-1,PATCH /v1/flexible-vector-databases/db-1,DELETE /v1/flexible-vector-databases/db-1"
	if strings.Join(calls, ",") != want {
		t.Fatalf("calls=%v", calls)
	}
}

func TestSmartBalancerLifecycle(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost, http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			routes, ok := body["routes"].([]any)
			if body["name"] != "router" || !ok || len(routes) != 1 {
				t.Errorf("body=%#v", body)
			}
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			}
			_, _ = io.WriteString(w, `{"id":"bal-1","name":"router","routes":[{"name":"default","destination":{"type":"instance","ref":"i-1"}}]}`)
		case http.MethodGet:
			_, _ = io.WriteString(w, `{"smart_balancers":[{"id":"bal-1","name":"router"}]}`)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	name := "default"
	routes := []SmartBalancerRoute{{Name: &name, Destination: &BalancerDestination{Type: "instance", Ref: "i-1"}}}
	input := SmartBalancerInput{Name: "router", Routes: &routes}
	created, err := client.CreateSmartBalancer(context.Background(), input)
	if err != nil || created.ID != "bal-1" {
		t.Fatalf("create=%#v error=%v", created, err)
	}
	if _, err := client.GetSmartBalancer(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetSmartBalancer(context.Background(), "missing"); !IsNotFound(err) {
		t.Fatalf("missing error=%v", err)
	}
	if _, err := client.UpdateSmartBalancer(context.Background(), created.ID, input); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteSmartBalancer(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	want := "POST /v1/smart-balancers,GET /v1/smart-balancers,GET /v1/smart-balancers,PATCH /v1/smart-balancers/bal-1,DELETE /v1/smart-balancers/bal-1"
	if strings.Join(calls, ",") != want {
		t.Fatalf("calls=%v", calls)
	}
}

func TestTargetGroupLifecycle(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if _, ok := body["members"].([]any); !ok {
				t.Errorf("missing target group members: %#v", body)
			}
			if r.Method == http.MethodPatch {
				if _, ok := body["name"]; ok {
					t.Error("name cannot be patched")
				}
				if _, ok := body["workload_kind"]; ok {
					t.Error("workload_kind cannot be patched")
				}
			}
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			}
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = io.WriteString(w, `{"id":"group-1","name":"workers","members":[{"instance_id":"i-1","weight":1,"priority":1,"enabled":true}]}`)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	members := []TargetGroupMember{{InstanceID: "i-1", Weight: 1, Priority: 1, Enabled: true}}
	input := TargetGroupInput{Name: "workers", Members: &members}
	created, err := client.CreateTargetGroup(context.Background(), input)
	if err != nil || created.ID != "group-1" {
		t.Fatalf("create=%#v error=%v", created, err)
	}
	if _, err := client.GetTargetGroup(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := client.UpdateTargetGroup(context.Background(), created.ID, input); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteTargetGroup(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "POST /v1/target-groups,GET /v1/target-groups/group-1,PATCH /v1/target-groups/group-1,DELETE /v1/target-groups/group-1" {
		t.Fatalf("calls=%v", calls)
	}
}

func TestSmartBalancerReadRejectsMissingList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetSmartBalancer(context.Background(), "bal-1")
	if err == nil || IsNotFound(err) {
		t.Fatalf("missing balancer list must not be treated as a deleted balancer: %v", err)
	}
}
