package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QDivZero/qdivzero-terraform/internal/qdivzero"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestInstanceResourceLifecycle(t *testing.T) {
	ctx := context.Background()
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost, http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			}
			_, _ = fmt.Fprintf(w, `{"id":"i-1","name":%q,"huggingface_repo_id":"org/model","state":"running"}`, body["name"])
		case http.MethodGet:
			_, _ = fmt.Fprint(w, `{"id":"i-1","name":"updated","huggingface_repo_id":"org/model","state":"running"}`)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	client, err := qdivzero.NewClient(qdivzero.Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	r := &instanceResource{client: client}
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	s := schemaResponse.Schema
	makePlan := func(name string) tfsdk.Plan {
		typ := s.Type().TerraformType(ctx).(tftypes.Object)
		values := make(map[string]tftypes.Value, len(typ.AttributeTypes))
		for key, fieldType := range typ.AttributeTypes {
			values[key] = tftypes.NewValue(fieldType, tftypes.UnknownValue)
		}
		values["name"] = tftypes.NewValue(tftypes.String, name)
		values["huggingface_repo_id"] = tftypes.NewValue(tftypes.String, "org/model")
		values["desired_state"] = tftypes.NewValue(tftypes.String, nil)
		values["wait_timeout_seconds"] = tftypes.NewValue(tftypes.Number, nil)
		values["replica_count"] = tftypes.NewValue(tftypes.Number, nil)
		return tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(typ, values)}
	}
	plan := makePlan("example")
	created := resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: plan.Raw}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", created.Diagnostics)
	}
	var state instanceModel
	if diags := created.State.Get(ctx, &state); diags.HasError() || state.ID.ValueString() != "i-1" || state.State.ValueString() != "running" {
		t.Fatalf("created state = %#v, diagnostics = %v", state, diags)
	}
	updatedPlan := makePlan("updated")
	// A planned update retains the known identifier.
	values := map[string]tftypes.Value{}
	if err := updatedPlan.Raw.As(&values); err != nil {
		t.Fatal(err)
	}
	values["id"] = tftypes.NewValue(tftypes.String, "i-1")
	updatedPlan.Raw = tftypes.NewValue(s.Type().TerraformType(ctx), values)
	updated := resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: updatedPlan.Raw}}
	r.Update(ctx, resource.UpdateRequest{Plan: updatedPlan, State: created.State}, &updated)
	if updated.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", updated.Diagnostics)
	}
	read := resource.ReadResponse{State: updated.State}
	r.Read(ctx, resource.ReadRequest{State: updated.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", read.Diagnostics)
	}
	deleted := resource.DeleteResponse{State: read.State}
	r.Delete(ctx, resource.DeleteRequest{State: read.State}, &deleted)
	if deleted.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %v", deleted.Diagnostics)
	}
	if fmt.Sprint(methods) != "[POST PATCH GET DELETE]" {
		t.Fatalf("HTTP methods = %v", methods)
	}
}

func TestDesiredStateUpdateStopsAndReadDetectsDrift(t *testing.T) {
	ctx := context.Background()
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/v1/instances/i-1/stop" {
			_, _ = fmt.Fprint(w, `{"id":"i-1","state":"stopped"}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"id":"i-1","name":"example","huggingface_repo_id":"org/model","state":"running"}`)
	}))
	defer server.Close()
	client, err := qdivzero.NewClient(qdivzero.Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	r := &instanceResource{client: client}
	var definition resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &definition)
	state := tfsdk.State{Schema: definition.Schema}
	initial := instanceModel{ID: types.StringValue("i-1"), Name: types.StringValue("example"), HuggingfaceRepoID: types.StringValue("org/model"), DesiredState: types.StringValue("running"), State: types.StringValue("running"), GPUPreferences: types.ListNull(gpuPreferenceType()), SmartRegions: types.ListNull(types.StringType), SupportedProtocols: types.ListNull(types.StringType)}
	if diags := state.Set(ctx, &initial); diags.HasError() {
		t.Fatal(diags)
	}
	proposed := state
	if diags := proposed.SetAttribute(ctx, path.Root("desired_state"), types.StringValue("stopped")); diags.HasError() {
		t.Fatal(diags)
	}
	plan := tfsdk.Plan{Schema: definition.Schema, Raw: proposed.Raw}
	result := resource.UpdateResponse{State: proposed}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &result)
	if result.Diagnostics.HasError() {
		t.Fatal(result.Diagnostics)
	}
	var after instanceModel
	if diags := result.State.Get(ctx, &after); diags.HasError() || after.State.ValueString() != "stopped" {
		t.Fatalf("state=%#v diagnostics=%v", after, diags)
	}
	read := resource.ReadResponse{State: result.State}
	r.Read(ctx, resource.ReadRequest{State: result.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	if diags := read.State.Get(ctx, &after); diags.HasError() || after.DesiredState.ValueString() != "running" {
		t.Fatalf("drift state=%#v diagnostics=%v", after, diags)
	}
	if fmt.Sprint(calls) != "[GET /v1/instances/i-1 POST /v1/instances/i-1/stop GET /v1/instances/i-1]" {
		t.Fatalf("calls=%v", calls)
	}
}

func TestReadExposesFailedStateAsDesiredStateDrift(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"id":"i-1","name":"example","huggingface_repo_id":"org/model","state":"failed"}`)
	}))
	defer server.Close()
	client, err := qdivzero.NewClient(qdivzero.Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	r := &instanceResource{client: client}
	var definition resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &definition)
	state := tfsdk.State{Schema: definition.Schema}
	initial := instanceModel{ID: types.StringValue("i-1"), Name: types.StringValue("example"), HuggingfaceRepoID: types.StringValue("org/model"), DesiredState: types.StringValue("running"), State: types.StringValue("running"), GPUPreferences: types.ListNull(gpuPreferenceType()), SmartRegions: types.ListNull(types.StringType), SupportedProtocols: types.ListNull(types.StringType)}
	if diags := state.Set(ctx, &initial); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var actual instanceModel
	if diags := response.State.Get(ctx, &actual); diags.HasError() || actual.DesiredState.ValueString() != "failed" {
		t.Fatalf("expected failure to be visible as drift: %#v diagnostics=%v", actual, diags)
	}
}

func TestInstanceReadRemovesMissingResource(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client, err := qdivzero.NewClient(qdivzero.Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	r := &instanceResource{client: client}
	var schemaResponse resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	s := schemaResponse.Schema
	state := tfsdk.State{Schema: s}
	if diags := state.Set(ctx, &instanceModel{ID: types.StringValue("missing"), GPUPreferences: types.ListNull(gpuPreferenceType()), SmartRegions: types.ListNull(types.StringType), SupportedProtocols: types.ListNull(types.StringType)}); diags.HasError() {
		t.Fatalf("state setup: %v", diags)
	}
	response := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &response)
	if response.Diagnostics.HasError() || !response.State.Raw.IsNull() {
		t.Fatalf("missing instance should be removed from state: %v", response.Diagnostics)
	}
}

func TestInstanceDeleteRejectsMissingID(t *testing.T) {
	ctx := context.Background()
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	client, err := qdivzero.NewClient(qdivzero.Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	r := &instanceResource{client: client}
	var definition resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &definition)
	state := tfsdk.State{Schema: definition.Schema}
	if diags := state.Set(ctx, &instanceModel{ID: types.StringNull(), GPUPreferences: types.ListNull(gpuPreferenceType()), SmartRegions: types.ListNull(types.StringType), SupportedProtocols: types.ListNull(types.StringType)}); diags.HasError() {
		t.Fatalf("state setup: %v", diags)
	}
	result := resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, &result)
	if !result.Diagnostics.HasError() || called {
		t.Fatalf("missing ID should not make an API request: diagnostics=%v called=%v", result.Diagnostics, called)
	}
}

func TestCreateKeepsInstanceIDWhenStartingFails(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/instances" {
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, `{"id":"i-created","name":"example","huggingface_repo_id":"org/model","state":"non_bootstrap"}`)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprint(w, `{"error":"capacity unavailable"}`)
	}))
	defer server.Close()
	client, err := qdivzero.NewClient(qdivzero.Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	r := &instanceResource{client: client}
	var definition resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &definition)
	s := definition.Schema
	typ := s.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value)
	for key, fieldType := range typ.AttributeTypes {
		values[key] = tftypes.NewValue(fieldType, tftypes.UnknownValue)
	}
	values["name"] = tftypes.NewValue(tftypes.String, "example")
	values["huggingface_repo_id"] = tftypes.NewValue(tftypes.String, "org/model")
	values["desired_state"] = tftypes.NewValue(tftypes.String, "running")
	values["wait_timeout_seconds"] = tftypes.NewValue(tftypes.Number, 1)
	values["replica_count"] = tftypes.NewValue(tftypes.Number, nil)
	plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(typ, values)}
	response := resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: plan.Raw}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("expected start failure")
	}
	var model instanceModel
	if diags := response.State.Get(ctx, &model); diags.HasError() || model.ID.ValueString() != "i-created" {
		t.Fatalf("created instance must remain in state: model=%#v diagnostics=%v", model, diags)
	}
}
