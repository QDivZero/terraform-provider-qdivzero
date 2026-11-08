// Package provider implements the QDivZero Terraform provider.
package provider

import (
	"context"
	"os"

	"github.com/QDivZero/qdivzero-terraform/internal/qdivzero"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &qdivzeroProvider{}

type qdivzeroProvider struct{ version string }

type providerModel struct {
	APIKey           types.String `tfsdk:"api_key"`
	APIURL           types.String `tfsdk:"api_url"`
	AccountID        types.String `tfsdk:"account_id"`
	PrivateBetaToken types.String `tfsdk:"private_beta_token"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &qdivzeroProvider{version: version} }
}

func (p *qdivzeroProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "qdivzero"
	resp.Version = p.version
}

func (p *qdivzeroProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage QDivZero resources using an API key.",
		Attributes: map[string]schema.Attribute{
			"api_key":            schema.StringAttribute{Optional: true, Sensitive: true, Description: "API key; defaults to QDIVZERO_API_KEY."},
			"api_url":            schema.StringAttribute{Optional: true, Description: "API base URL; defaults to QDIVZERO_API_URL or https://api.qdiv0.com."},
			"account_id":         schema.StringAttribute{Optional: true, Description: "Account ID; defaults to QDIVZERO_ACCOUNT_ID."},
			"private_beta_token": schema.StringAttribute{Optional: true, Sensitive: true, Description: "Private beta token; defaults to QDIVZERO_PRIVATE_BETA_TOKEN."},
		},
	}
}

func (p *qdivzeroProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for name, value := range map[string]types.String{
		"api_key": data.APIKey, "api_url": data.APIURL,
		"account_id": data.AccountID, "private_beta_token": data.PrivateBetaToken,
	} {
		if value.IsUnknown() {
			resp.Diagnostics.AddError("Unknown provider configuration", "The "+name+" value must be known when configuring the provider.")
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}
	get := func(value types.String, env string) string {
		if !value.IsNull() {
			return value.ValueString()
		}
		return os.Getenv(env)
	}
	client, err := qdivzero.NewClient(qdivzero.Config{
		APIKey:           get(data.APIKey, "QDIVZERO_API_KEY"),
		BaseURL:          get(data.APIURL, "QDIVZERO_API_URL"),
		AccountID:        get(data.AccountID, "QDIVZERO_ACCOUNT_ID"),
		PrivateBetaToken: get(data.PrivateBetaToken, "QDIVZERO_PRIVATE_BETA_TOKEN"),
	})
	if err != nil {
		resp.Diagnostics.AddError("Invalid QDivZero provider configuration", err.Error())
		return
	}
	resp.ResourceData, resp.DataSourceData = client, client
}

func (p *qdivzeroProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{newInstanceResource, newScheduleRuleResource, newVectorDatabaseResource, newSmartBalancerResource, newTargetGroupResource}
}

func (p *qdivzeroProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{newServingEndpointsDataSource, newGPUsDataSource}
}
