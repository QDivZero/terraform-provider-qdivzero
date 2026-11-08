package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QDivZero/qdivzero-terraform/internal/qdivzero"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestSmartBalancerResourceLifecycle(t *testing.T) {
	ctx := context.Background()
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		object := `{"id":"bal-1","name":"router","routes":[{"name":"default","intent":"","priority":1,"enabled":true,"is_default":true,"destination":{"type":"instance","ref":"i-1"}}]}`
		switch r.Method {
		case http.MethodPost, http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			routes, ok := body["routes"].([]any)
			if !ok || len(routes) != 1 {
				t.Errorf("routes=%#v", body["routes"])
			}
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			}
			_, _ = fmt.Fprint(w, object)
		case http.MethodGet:
			_, _ = fmt.Fprintf(w, `{"smart_balancers":[%s]}`, object)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	client, err := qdivzero.NewClient(qdivzero.Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	r := &smartBalancerResource{client: client}
	var definition resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &definition)
	s := definition.Schema
	typ := s.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value)
	for key, fieldType := range typ.AttributeTypes {
		values[key] = tftypes.NewValue(fieldType, tftypes.UnknownValue)
	}
	values["name"] = tftypes.NewValue(tftypes.String, "router")
	routesType := typ.AttributeTypes["routes"].(tftypes.List)
	route := tftypes.NewValue(routesType.ElementType, map[string]tftypes.Value{
		"name": tftypes.NewValue(tftypes.String, "default"), "intent": tftypes.NewValue(tftypes.String, ""),
		"priority": tftypes.NewValue(tftypes.Number, 1), "enabled": tftypes.NewValue(tftypes.Bool, true),
		"is_default": tftypes.NewValue(tftypes.Bool, true), "destination_type": tftypes.NewValue(tftypes.String, "instance"),
		"destination_ref": tftypes.NewValue(tftypes.String, "i-1"),
	})
	values["routes"] = tftypes.NewValue(routesType, []tftypes.Value{route})
	plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(typ, values)}
	created := resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: plan.Raw}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	var model smartBalancerModel
	if diags := created.State.Get(ctx, &model); diags.HasError() || model.ID.ValueString() != "bal-1" {
		t.Fatalf("state=%#v diagnostics=%v", model, diags)
	}
	read := resource.ReadResponse{State: created.State}
	r.Read(ctx, resource.ReadRequest{State: created.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	if err := plan.Raw.As(&values); err != nil {
		t.Fatal(err)
	}
	values["id"] = tftypes.NewValue(tftypes.String, "bal-1")
	plan.Raw = tftypes.NewValue(typ, values)
	updated := resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: plan.Raw}}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: read.State}, &updated)
	if updated.Diagnostics.HasError() {
		t.Fatal(updated.Diagnostics)
	}
	deleted := resource.DeleteResponse{State: updated.State}
	r.Delete(ctx, resource.DeleteRequest{State: updated.State}, &deleted)
	if deleted.Diagnostics.HasError() {
		t.Fatal(deleted.Diagnostics)
	}
	if fmt.Sprint(calls) != "[POST /v1/smart-balancers GET /v1/smart-balancers PATCH /v1/smart-balancers/bal-1 DELETE /v1/smart-balancers/bal-1]" {
		t.Fatalf("calls=%v", calls)
	}
}

func TestSmartBalancerReadAcceptsOmittedOptionalRouteFields(t *testing.T) {
	name := "route"
	priority := int64(1)
	routes := []qdivzero.SmartBalancerRoute{{Name: &name, Priority: &priority, Destination: &qdivzero.BalancerDestination{Type: "instance", Ref: "i-1"}}}
	model := smartBalancerModel{Routes: types.ListNull(smartRouteType())}
	if diagnostics := model.apply(context.Background(), qdivzero.SmartBalancer{ID: "bal-1", Routes: &routes}, true); diagnostics.HasError() {
		t.Fatalf("optional route fields should be accepted: %v", diagnostics)
	}
	var values []smartBalancerRouteModel
	if diagnostics := model.Routes.ElementsAs(context.Background(), &values, false); diagnostics.HasError() || len(values) != 1 {
		t.Fatalf("routes=%#v diagnostics=%v", values, diagnostics)
	}
}

func TestSmartBalancerValidationRequiresRouterModelPair(t *testing.T) {
	ctx := context.Background()
	r := &smartBalancerResource{}
	var definition resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &definition)
	state := tfsdk.State{Schema: definition.Schema}
	model := smartBalancerModel{Name: types.StringValue("router"), RouterModelType: types.StringValue("public_model"), Routes: types.ListNull(smartRouteType())}
	if diags := state.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	result := resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: definition.Schema, Raw: state.Raw}}, &result)
	if !result.Diagnostics.HasError() {
		t.Fatal("a router model type without a reference must be rejected")
	}
}
