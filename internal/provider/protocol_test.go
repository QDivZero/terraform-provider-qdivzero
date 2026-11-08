package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProtocolExposesValidSchemas(t *testing.T) {
	server := providerserver.NewProtocol6(New("test")())()
	response, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("schema diagnostic: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
	for _, name := range []string{"qdivzero_instance", "qdivzero_instance_schedule", "qdivzero_vector_database", "qdivzero_smart_balancer", "qdivzero_target_group"} {
		if response.ResourceSchemas[name] == nil {
			t.Errorf("provider protocol is missing resource %s", name)
		}
	}
	if response.DataSourceSchemas["qdivzero_serving_endpoints"] == nil {
		t.Fatal("provider protocol is missing the serving endpoints data source")
	}
	if response.DataSourceSchemas["qdivzero_gpus"] == nil {
		t.Fatal("provider protocol is missing the GPU catalog data source")
	}
}

func TestProviderProtocolWithMockAPI(t *testing.T) {
	ctx := context.Background()
	var calls []string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("API key was not sent as a bearer credential")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/instances":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["scheduling_mode"] != "smart" || body["smart_max_price_per_hour_eur"] != 0.5 || body["smart_allow_spot"] != true || body["smart_allow_community"] != true || body["serverless_enabled"] != true || body["workload_kind"] != "decision" {
				t.Errorf("unexpected Smart configuration: %#v", body)
			}
			if regions, ok := body["smart_regions"].([]any); !ok || len(regions) != 2 || regions[0] != "Europe" || regions[1] != "North America" {
				t.Errorf("unexpected Smart regions: %#v", body["smart_regions"])
			}
			if preferences, ok := body["gpu_preferences"].([]any); !ok || len(preferences) != 1 {
				t.Errorf("missing GPU preferences: %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, `{"id":"i-1","name":"example","huggingface_repo_id":"org/model","description":"server-default","state":"non_bootstrap"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/instances/i-1/start":
			_, _ = fmt.Fprint(w, `{"id":"i-1","state":"running"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/instances/i-1":
			_, _ = fmt.Fprint(w, `{"id":"i-1","name":"example","huggingface_repo_id":"org/model","state":"running"}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/instances/i-1":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/serving-endpoints":
			_, _ = fmt.Fprint(w, `{"endpoints":[{"id":"ep-1","name":"model","state":"ready"}]}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer api.Close()
	server := providerserver.NewProtocol6(New("test")())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, schemas.Diagnostics)
	dynamic := func(typ tftypes.Type, value tftypes.Value) *tfprotov6.DynamicValue {
		t.Helper()
		result, err := tfprotov6.NewDynamicValue(typ, value)
		if err != nil {
			t.Fatal(err)
		}
		return &result
	}
	providerType := schemas.Provider.ValueType()
	providerConfig := dynamic(providerType, tftypes.NewValue(providerType, map[string]tftypes.Value{
		"api_key":            tftypes.NewValue(tftypes.String, "test-key"),
		"api_url":            tftypes.NewValue(tftypes.String, api.URL),
		"account_id":         tftypes.NewValue(tftypes.String, nil),
		"private_beta_token": tftypes.NewValue(tftypes.String, nil),
	}))
	configured, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: providerConfig})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, configured.Diagnostics)
	instanceType := schemas.ResourceSchemas["qdivzero_instance"].ValueType()
	instanceFields := make(map[string]tftypes.Value)
	for key, fieldType := range instanceType.(tftypes.Object).AttributeTypes {
		instanceFields[key] = tftypes.NewValue(fieldType, nil)
	}
	instanceFields["name"] = tftypes.NewValue(tftypes.String, "example")
	instanceFields["huggingface_repo_id"] = tftypes.NewValue(tftypes.String, "org/model")
	instanceFields["smart_regions"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "Europe"), tftypes.NewValue(tftypes.String, "North America")})
	instanceFields["smart_max_price_per_hour_eur"] = tftypes.NewValue(tftypes.Number, 0.5)
	instanceFields["smart_allow_spot"] = tftypes.NewValue(tftypes.Bool, true)
	instanceFields["smart_allow_community"] = tftypes.NewValue(tftypes.Bool, true)
	instanceFields["serverless_enabled"] = tftypes.NewValue(tftypes.Bool, true)
	instanceFields["workload_kind"] = tftypes.NewValue(tftypes.String, "decision")
	instanceFields["supported_protocols"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "openai")})
	instanceFields["desired_state"] = tftypes.NewValue(tftypes.String, "running")
	instanceFields["wait_timeout_seconds"] = tftypes.NewValue(tftypes.Number, 5)
	preferencesType := instanceType.(tftypes.Object).AttributeTypes["gpu_preferences"].(tftypes.List)
	preference := tftypes.NewValue(preferencesType.ElementType, map[string]tftypes.Value{
		"public_gpu_id": tftypes.NewValue(tftypes.String, "gpu-1"),
		"count":         tftypes.NewValue(tftypes.Number, 1),
	})
	instanceFields["gpu_preferences"] = tftypes.NewValue(preferencesType, []tftypes.Value{preference})
	instanceConfig := dynamic(instanceType, tftypes.NewValue(instanceType, instanceFields))
	configValue, err := instanceConfig.Unmarshal(instanceType)
	if err != nil {
		t.Fatal(err)
	}
	var invalidFields map[string]tftypes.Value
	if err := configValue.As(&invalidFields); err != nil {
		t.Fatal(err)
	}
	invalidFields["name"] = tftypes.NewValue(tftypes.String, " ")
	invalidConfig := dynamic(instanceType, tftypes.NewValue(instanceType, invalidFields))
	validation, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
		TypeName: "qdivzero_instance", Config: invalidConfig,
	})
	if err != nil {
		t.Fatal(err)
	}
	hasError := false
	for _, diagnostic := range validation.Diagnostics {
		hasError = hasError || diagnostic.Severity == tfprotov6.DiagnosticSeverityError
	}
	if !hasError {
		t.Fatal("blank instance name must be rejected")
	}
	emptyState := dynamic(instanceType, tftypes.NewValue(instanceType, nil))
	planned, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName: "qdivzero_instance", PriorState: emptyState, ProposedNewState: instanceConfig, Config: instanceConfig,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, planned.Diagnostics)
	created, err := server.ApplyResourceChange(ctx, &tfprotov6.ApplyResourceChangeRequest{
		TypeName: "qdivzero_instance", PriorState: emptyState, PlannedState: planned.PlannedState, Config: instanceConfig,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, created.Diagnostics)
	createdValue, err := created.NewState.Unmarshal(instanceType)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]tftypes.Value
	if err := createdValue.As(&fields); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := fields["id"].As(&id); err != nil || id != "i-1" {
		t.Fatalf("instance ID = %q, error = %v", id, err)
	}
	var description string
	if err := fields["description"].As(&description); err != nil || description != "server-default" {
		t.Fatalf("computed description = %q, error = %v", description, err)
	}
	fields["huggingface_repo_id"] = tftypes.NewValue(tftypes.String, "other/model")
	changed := dynamic(instanceType, tftypes.NewValue(instanceType, fields))
	replacement, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName: "qdivzero_instance", PriorState: created.NewState, ProposedNewState: changed, Config: changed,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, replacement.Diagnostics)
	if len(replacement.RequiresReplace) != 1 {
		t.Fatalf("changing the model repository must require replacement: %v", replacement.RequiresReplace)
	}
	read, err := server.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: "qdivzero_instance", CurrentState: created.NewState})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, read.Diagnostics)
	dataType := schemas.DataSourceSchemas["qdivzero_serving_endpoints"].ValueType()
	dataConfig := dynamic(dataType, tftypes.NewValue(dataType, map[string]tftypes.Value{
		"endpoints": tftypes.NewValue(dataType.(tftypes.Object).AttributeTypes["endpoints"], nil),
	}))
	listed, err := server.ReadDataSource(ctx, &tfprotov6.ReadDataSourceRequest{TypeName: "qdivzero_serving_endpoints", Config: dataConfig})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, listed.Diagnostics)
	deleted, err := server.ApplyResourceChange(ctx, &tfprotov6.ApplyResourceChangeRequest{
		TypeName: "qdivzero_instance", PriorState: read.NewState, PlannedState: emptyState, Config: emptyState,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, deleted.Diagnostics)
	if fmt.Sprint(calls) != "[POST /v1/instances POST /v1/instances/i-1/start GET /v1/instances/i-1 GET /v1/serving-endpoints DELETE /v1/instances/i-1]" {
		t.Fatalf("unexpected API calls: %v", calls)
	}
}

func assertProtocolDiagnostics(t *testing.T, diagnostics []*tfprotov6.Diagnostic) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("protocol diagnostic: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
}

func TestSmartBalancerProtocolAppliesRouteDefaults(t *testing.T) {
	ctx := context.Background()
	called := false
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		var body struct {
			Routes []struct {
				Intent    string `json:"intent"`
				Enabled   bool   `json:"enabled"`
				IsDefault bool   `json:"is_default"`
			} `json:"routes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Routes) != 1 || body.Routes[0].Intent != "" || !body.Routes[0].Enabled || body.Routes[0].IsDefault {
			t.Errorf("route defaults not sent correctly: %#v", body.Routes)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"id":"bal-1","name":"router"}`)
	}))
	defer api.Close()
	server := providerserver.NewProtocol6(New("test")())()
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, schemas.Diagnostics)
	dynamic := func(typ tftypes.Type, value tftypes.Value) *tfprotov6.DynamicValue {
		t.Helper()
		result, err := tfprotov6.NewDynamicValue(typ, value)
		if err != nil {
			t.Fatal(err)
		}
		return &result
	}
	providerType := schemas.Provider.ValueType()
	providerConfig := dynamic(providerType, tftypes.NewValue(providerType, map[string]tftypes.Value{
		"api_key": tftypes.NewValue(tftypes.String, "key"), "api_url": tftypes.NewValue(tftypes.String, api.URL),
		"account_id": tftypes.NewValue(tftypes.String, nil), "private_beta_token": tftypes.NewValue(tftypes.String, nil),
	}))
	configured, err := server.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{Config: providerConfig})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, configured.Diagnostics)
	typ := schemas.ResourceSchemas["qdivzero_smart_balancer"].ValueType()
	values := make(map[string]tftypes.Value)
	for name, fieldType := range typ.(tftypes.Object).AttributeTypes {
		values[name] = tftypes.NewValue(fieldType, nil)
	}
	values["name"] = tftypes.NewValue(tftypes.String, "router")
	routesType := typ.(tftypes.Object).AttributeTypes["routes"].(tftypes.List)
	route := tftypes.NewValue(routesType.ElementType, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "primary"), "intent": tftypes.NewValue(tftypes.String, nil),
		"priority": tftypes.NewValue(tftypes.Number, 1), "enabled": tftypes.NewValue(tftypes.Bool, nil),
		"is_default": tftypes.NewValue(tftypes.Bool, nil), "destination_type": tftypes.NewValue(tftypes.String, "instance"),
		"destination_ref": tftypes.NewValue(tftypes.String, "i-1"),
	})
	values["routes"] = tftypes.NewValue(routesType, []tftypes.Value{route})
	config := dynamic(typ, tftypes.NewValue(typ, values))
	empty := dynamic(typ, tftypes.NewValue(typ, nil))
	plan, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{TypeName: "qdivzero_smart_balancer", PriorState: empty, ProposedNewState: config, Config: config})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, plan.Diagnostics)
	created, err := server.ApplyResourceChange(ctx, &tfprotov6.ApplyResourceChangeRequest{TypeName: "qdivzero_smart_balancer", PriorState: empty, PlannedState: plan.PlannedState, Config: config})
	if err != nil {
		t.Fatal(err)
	}
	assertProtocolDiagnostics(t, created.Diagnostics)
	if !called {
		t.Fatal("expected a request to the mock API")
	}
}
