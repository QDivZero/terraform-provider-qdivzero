package provider

import (
	"context"
	"fmt"

	"github.com/QDivZero/qdivzero-terraform/internal/qdivzero"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &servingEndpointsDataSource{}
var _ datasource.DataSourceWithConfigure = &servingEndpointsDataSource{}

type servingEndpointsDataSource struct{ client *qdivzero.Client }

type servingEndpointsModel struct {
	Endpoints types.List `tfsdk:"endpoints"`
}

type servingEndpointModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	State       types.String `tfsdk:"state"`
	Enabled     types.Bool   `tfsdk:"enabled"`
}

func newServingEndpointsDataSource() datasource.DataSource { return &servingEndpointsDataSource{} }

func (d *servingEndpointsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_serving_endpoints"
}

func (d *servingEndpointsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "List serving endpoints visible to the configured QDivZero account.",
		Attributes: map[string]schema.Attribute{
			"endpoints": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":           schema.StringAttribute{Computed: true},
					"name":         schema.StringAttribute{Computed: true},
					"display_name": schema.StringAttribute{Computed: true},
					"state":        schema.StringAttribute{Computed: true},
					"enabled":      schema.BoolAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *servingEndpointsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*qdivzero.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected a QDivZero client, got %T.", req.ProviderData))
		return
	}
	d.client = client
}

func (d *servingEndpointsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Provider not configured", "Configure the QDivZero provider before listing serving endpoints.")
		return
	}
	endpoints, err := d.client.ListServingEndpoints(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list serving endpoints", err.Error())
		return
	}
	items := make([]servingEndpointModel, 0, len(endpoints))
	for _, endpoint := range endpoints {
		items = append(items, servingEndpointModel{
			ID: types.StringValue(endpoint.ID), Name: types.StringPointerValue(endpoint.Name),
			DisplayName: types.StringPointerValue(endpoint.DisplayName), State: types.StringPointerValue(endpoint.State),
			Enabled: types.BoolPointerValue(endpoint.Enabled),
		})
	}
	objectType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id": types.StringType, "name": types.StringType, "display_name": types.StringType,
		"state": types.StringType, "enabled": types.BoolType,
	}}
	list, diags := types.ListValueFrom(ctx, objectType, items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &servingEndpointsModel{Endpoints: list})...)
}
