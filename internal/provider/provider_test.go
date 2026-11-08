package provider

import (
	"context"
	"testing"

	framework "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProviderSchemaAndRegistration(t *testing.T) {
	ctx := context.Background()
	p := New("test")()
	var metadata framework.MetadataResponse
	p.Metadata(ctx, framework.MetadataRequest{}, &metadata)
	if metadata.TypeName != "qdivzero" || metadata.Version != "test" {
		t.Fatalf("unexpected provider metadata: %#v", metadata)
	}
	var response framework.SchemaResponse
	p.Schema(ctx, framework.SchemaRequest{}, &response)
	key, ok := response.Schema.Attributes["api_key"].(schema.StringAttribute)
	if !ok || !key.Sensitive || !key.Optional {
		t.Fatalf("api_key should be an optional sensitive attribute: %#v", key)
	}
	if len(p.Resources(ctx)) == 0 || len(p.DataSources(ctx)) == 0 {
		t.Fatal("provider must register its resource and data source")
	}
}

func TestProviderConfigureUsesEnvironmentAndRejectsUnknownKey(t *testing.T) {
	ctx := context.Background()
	p := New("test")()
	var definition framework.SchemaResponse
	p.Schema(ctx, framework.SchemaRequest{}, &definition)
	tfType := definition.Schema.Type().TerraformType(ctx)
	config := func(apiKey tftypes.Value) tfsdk.Config {
		return tfsdk.Config{Schema: definition.Schema, Raw: tftypes.NewValue(tfType, map[string]tftypes.Value{
			"api_key":            apiKey,
			"api_url":            tftypes.NewValue(tftypes.String, nil),
			"account_id":         tftypes.NewValue(tftypes.String, nil),
			"private_beta_token": tftypes.NewValue(tftypes.String, nil),
		})}
	}
	t.Setenv("QDIVZERO_API_KEY", "environment-key")
	t.Setenv("QDIVZERO_API_URL", "")
	t.Setenv("QDIVZERO_ACCOUNT_ID", "")
	t.Setenv("QDIVZERO_PRIVATE_BETA_TOKEN", "")
	var configured framework.ConfigureResponse
	p.Configure(ctx, framework.ConfigureRequest{Config: config(tftypes.NewValue(tftypes.String, nil))}, &configured)
	if configured.Diagnostics.HasError() || configured.ResourceData == nil || configured.DataSourceData == nil {
		t.Fatalf("provider configuration failed: %v", configured.Diagnostics)
	}
	var unknown framework.ConfigureResponse
	p.Configure(ctx, framework.ConfigureRequest{Config: config(tftypes.NewValue(tftypes.String, tftypes.UnknownValue))}, &unknown)
	if !unknown.Diagnostics.HasError() {
		t.Fatal("unknown API key must produce a diagnostic")
	}
}
