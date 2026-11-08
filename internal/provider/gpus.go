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

var _ datasource.DataSource = &gpusDataSource{}
var _ datasource.DataSourceWithConfigure = &gpusDataSource{}

type gpusDataSource struct{ client *qdivzero.Client }

type gpusModel struct {
	GPUs types.List `tfsdk:"gpus"`
}

type gpuModel struct {
	PublicGPUID       types.String  `tfsdk:"public_gpu_id"`
	DisplayName       types.String  `tfsdk:"display_name"`
	ProviderCodename  types.String  `tfsdk:"provider_codename"`
	MarketType        types.String  `tfsdk:"market_type"`
	Continent         types.String  `tfsdk:"continent"`
	Country           types.String  `tfsdk:"country"`
	City              types.String  `tfsdk:"city"`
	PricePerHourEUR   types.Float64 `tfsdk:"price_per_hour_eur"`
	VRAMGB            types.Float64 `tfsdk:"vram_gb"`
	GPUCount          types.Int64   `tfsdk:"gpu_count"`
	AvailabilityCount types.Int64   `tfsdk:"availability_count"`
	StockStatus       types.String  `tfsdk:"stock_status"`
}

func newGPUsDataSource() datasource.DataSource { return &gpusDataSource{} }

func (d *gpusDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gpus"
}

func (d *gpusDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "List available GPUs, prices, regions, and market types.", Attributes: map[string]schema.Attribute{
		"gpus": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"public_gpu_id": schema.StringAttribute{Computed: true}, "display_name": schema.StringAttribute{Computed: true},
			"provider_codename": schema.StringAttribute{Computed: true}, "market_type": schema.StringAttribute{Computed: true},
			"continent": schema.StringAttribute{Computed: true}, "country": schema.StringAttribute{Computed: true}, "city": schema.StringAttribute{Computed: true},
			"price_per_hour_eur": schema.Float64Attribute{Computed: true}, "vram_gb": schema.Float64Attribute{Computed: true},
			"gpu_count": schema.Int64Attribute{Computed: true}, "availability_count": schema.Int64Attribute{Computed: true}, "stock_status": schema.StringAttribute{Computed: true},
		}}},
	}}
}

func (d *gpusDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *gpusDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Provider not configured", "A configured QDivZero provider is required.")
		return
	}
	gpus, err := d.client.ListGPUs(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list GPUs", err.Error())
		return
	}
	items := make([]gpuModel, 0, len(gpus))
	for _, gpu := range gpus {
		items = append(items, gpuModel{
			PublicGPUID: types.StringValue(gpu.PublicGPUID), DisplayName: types.StringPointerValue(gpu.DisplayName),
			ProviderCodename: types.StringPointerValue(gpu.ProviderCodename), MarketType: types.StringPointerValue(gpu.MarketType),
			Continent: types.StringPointerValue(gpu.Continent), Country: types.StringPointerValue(gpu.Country), City: types.StringPointerValue(gpu.City),
			PricePerHourEUR: types.Float64PointerValue(gpu.PricePerHourEUR), VRAMGB: types.Float64PointerValue(gpu.VRAMGB),
			GPUCount: types.Int64PointerValue(gpu.GPUCount), AvailabilityCount: types.Int64PointerValue(gpu.AvailabilityCount), StockStatus: types.StringPointerValue(gpu.StockStatus),
		})
	}
	objectType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"public_gpu_id": types.StringType, "display_name": types.StringType, "provider_codename": types.StringType, "market_type": types.StringType,
		"continent": types.StringType, "country": types.StringType, "city": types.StringType, "price_per_hour_eur": types.Float64Type,
		"vram_gb": types.Float64Type, "gpu_count": types.Int64Type, "availability_count": types.Int64Type, "stock_status": types.StringType,
	}}
	value, diags := types.ListValueFrom(ctx, objectType, items)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &gpusModel{GPUs: value})...)
	}
}
