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

func TestGPUsDataSource(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"gpus":[{"public_gpu_id":"gpu-b","price_per_hour_eur":1.2},{"public_gpu_id":"gpu-a","market_type":"spot","continent":"Europe"}]}`)
	}))
	defer server.Close()
	client, err := qdivzero.NewClient(qdivzero.Config{BaseURL: server.URL, APIKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	d := &gpusDataSource{client: client}
	var definition datasource.SchemaResponse
	d.Schema(ctx, datasource.SchemaRequest{}, &definition)
	response := datasource.ReadResponse{State: tfsdk.State{Schema: definition.Schema}}
	d.Read(ctx, datasource.ReadRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var state gpusModel
	if diags := response.State.Get(ctx, &state); diags.HasError() {
		t.Fatal(diags)
	}
	var gpus []gpuModel
	if diags := state.GPUs.ElementsAs(ctx, &gpus, false); diags.HasError() {
		t.Fatal(diags)
	}
	if len(gpus) != 2 || gpus[0].PublicGPUID.ValueString() != "gpu-a" || gpus[0].MarketType.ValueString() != "spot" {
		t.Fatalf("unexpected GPUs: %#v", gpus)
	}
}
