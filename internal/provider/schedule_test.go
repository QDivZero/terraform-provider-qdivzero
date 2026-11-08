package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QDivZero/qdivzero-terraform/internal/qdivzero"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestScheduleResourceLifecycle(t *testing.T) {
	ctx := context.Background()
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost, http.MethodPatch:
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			}
			_, _ = fmt.Fprint(w, `{"id":"rule-1","instance_id":"i-1","action":"start","cron":"0 9 * * 1-5","timezone":"Europe/Madrid","enabled":true}`)
		case http.MethodGet:
			_, _ = fmt.Fprint(w, `{"schedule_rules":[{"id":"rule-1","instance_id":"i-1","action":"start","cron":"0 9 * * 1-5","timezone":"Europe/Madrid","enabled":true}]}`)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	client, err := qdivzero.NewClient(qdivzero.Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	r := &scheduleRuleResource{client: client}
	var definition resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &definition)
	s := definition.Schema
	typ := s.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value)
	for key, fieldType := range typ.AttributeTypes {
		values[key] = tftypes.NewValue(fieldType, tftypes.UnknownValue)
	}
	values["instance_id"] = tftypes.NewValue(tftypes.String, "i-1")
	values["action"] = tftypes.NewValue(tftypes.String, "start")
	values["cron"] = tftypes.NewValue(tftypes.String, "0 9 * * 1-5")
	plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(typ, values)}
	created := resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: plan.Raw}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	var model scheduleRuleModel
	if diags := created.State.Get(ctx, &model); diags.HasError() || model.ID.ValueString() != "rule-1" {
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
	values["id"] = tftypes.NewValue(tftypes.String, "rule-1")
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
	if fmt.Sprint(calls) != "[POST /v1/instances/i-1/schedule-rules GET /v1/instances/i-1/schedule-rules PATCH /v1/instances/i-1/schedule-rules/rule-1 DELETE /v1/instances/i-1/schedule-rules/rule-1]" {
		t.Fatalf("calls=%v", calls)
	}
}

func TestScheduleImportUsesInstanceAndRuleIDs(t *testing.T) {
	ctx := context.Background()
	r := &scheduleRuleResource{}
	var definition resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &definition)
	s := definition.Schema
	state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	response := resource.ImportStateResponse{State: state}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "i-1/rule-1"}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var model scheduleRuleModel
	if diags := response.State.Get(ctx, &model); diags.HasError() || model.InstanceID.ValueString() != "i-1" || model.ID.ValueString() != "rule-1" {
		t.Fatalf("state=%#v diagnostics=%v", model, diags)
	}
	invalid := resource.ImportStateResponse{State: state}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "rule-only"}, &invalid)
	if !invalid.Diagnostics.HasError() {
		t.Fatal("invalid composite ID must fail")
	}
}
