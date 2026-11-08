package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QDivZero/qdivzero-terraform/internal/qdivzero"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

func TestServingEndpointsDataSourceReadsSortedResults(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"first":[{"id":"b","name":"second"}],"second":[{"id":"a","name":"first","enabled":true}]}`)
	}))
	defer server.Close()
	client, err := qdivzero.NewClient(qdivzero.Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	d := &servingEndpointsDataSource{client: client}
	var schemaResponse datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &schemaResponse)
	response := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
	d.Read(ctx, datasource.ReadRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", response.Diagnostics)
	}
	var model servingEndpointsModel
	if diags := response.State.Get(ctx, &model); diags.HasError() {
		t.Fatalf("state diagnostics: %v", diags)
	}
	var endpoints []servingEndpointModel
	if diags := model.Endpoints.ElementsAs(ctx, &endpoints, false); diags.HasError() {
		t.Fatalf("endpoints diagnostics: %v", diags)
	}
	if len(endpoints) != 2 || endpoints[0].ID.ValueString() != "a" || endpoints[1].ID.ValueString() != "b" {
		t.Fatalf("unexpected endpoints: %#v", endpoints)
	}
}
