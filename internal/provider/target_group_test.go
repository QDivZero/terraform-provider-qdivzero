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

func TestTargetGroupResourceLifecycle(t *testing.T) {
	ctx := context.Background()
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = fmt.Fprint(w, `{"id":"g-1","name":"workers","members":[{"instance_id":"i-1","weight":1,"priority":1,"enabled":true}]}`)
	}))
	defer server.Close()
	client, err := qdivzero.NewClient(qdivzero.Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	r := &targetGroupResource{client: client}
	var definition resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &definition)
	s := definition.Schema
	typ := s.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value)
	for key, fieldType := range typ.AttributeTypes {
		values[key] = tftypes.NewValue(fieldType, tftypes.UnknownValue)
	}
	values["name"] = tftypes.NewValue(tftypes.String, "workers")
	membersType := typ.AttributeTypes["members"].(tftypes.List)
	member := tftypes.NewValue(membersType.ElementType, map[string]tftypes.Value{
		"instance_id": tftypes.NewValue(tftypes.String, "i-1"), "weight": tftypes.NewValue(tftypes.Number, 1),
		"priority": tftypes.NewValue(tftypes.Number, 1), "enabled": tftypes.NewValue(tftypes.Bool, true),
	})
	values["members"] = tftypes.NewValue(membersType, []tftypes.Value{member})
	plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(typ, values)}
	created := resource.CreateResponse{State: tfsdk.State{Schema: s, Raw: plan.Raw}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &created)
	if created.Diagnostics.HasError() {
		t.Fatal(created.Diagnostics)
	}
	var model targetGroupModel
	if diags := created.State.Get(ctx, &model); diags.HasError() || model.ID.ValueString() != "g-1" {
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
	values["id"] = tftypes.NewValue(tftypes.String, "g-1")
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
	if fmt.Sprint(calls) != "[POST /v1/target-groups GET /v1/target-groups/g-1 PATCH /v1/target-groups/g-1 DELETE /v1/target-groups/g-1]" {
		t.Fatalf("calls=%v", calls)
	}
}
