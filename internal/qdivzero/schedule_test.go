package qdivzero

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScheduleRuleLifecycle(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost, http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["action"] != "start" || body["cron"] != "0 9 * * 1-5" {
				t.Errorf("unexpected schedule body: %#v", body)
			}
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			}
			_, _ = io.WriteString(w, `{"id":"rule-1","instance_id":"i-1","action":"start","cron":"0 9 * * 1-5","timezone":"Europe/Madrid","enabled":true}`)
		case http.MethodGet:
			_, _ = io.WriteString(w, `{"schedule_rules":[{"id":"rule-1","instance_id":"i-1","action":"start","cron":"0 9 * * 1-5"}]}`)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	zone := "Europe/Madrid"
	input := ScheduleRuleInput{Action: "start", Cron: "0 9 * * 1-5", Timezone: &zone}
	created, err := client.CreateScheduleRule(context.Background(), "i-1", input)
	if err != nil || created.ID != "rule-1" {
		t.Fatalf("create=%#v error=%v", created, err)
	}
	if _, err := client.GetScheduleRule(context.Background(), "i-1", "rule-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetScheduleRule(context.Background(), "i-1", "missing"); !IsNotFound(err) {
		t.Fatalf("missing rule error=%v", err)
	}
	if _, err := client.UpdateScheduleRule(context.Background(), "i-1", "rule-1", input); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteScheduleRule(context.Background(), "i-1", "rule-1"); err != nil {
		t.Fatal(err)
	}
	want := []string{"POST /v1/instances/i-1/schedule-rules", "GET /v1/instances/i-1/schedule-rules", "GET /v1/instances/i-1/schedule-rules", "PATCH /v1/instances/i-1/schedule-rules/rule-1", "DELETE /v1/instances/i-1/schedule-rules/rule-1"}
	if len(requests) != len(want) {
		t.Fatalf("requests=%v", requests)
	}
	for i := range want {
		if requests[i] != want[i] {
			t.Fatalf("requests=%v", requests)
		}
	}
}

func TestScheduleReadRejectsMissingList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()
	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetScheduleRule(context.Background(), "i-1", "rule-1")
	if err == nil || IsNotFound(err) {
		t.Fatalf("missing schedule list must not be treated as a deleted rule: %v", err)
	}
}
