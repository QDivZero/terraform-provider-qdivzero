package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/QDivZero/qdivzero-terraform/internal/qdivzero"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &smartBalancerResource{}
var _ resource.ResourceWithConfigure = &smartBalancerResource{}
var _ resource.ResourceWithImportState = &smartBalancerResource{}
var _ resource.ResourceWithValidateConfig = &smartBalancerResource{}

type smartBalancerResource struct{ client *qdivzero.Client }

type smartBalancerModel struct {
	ID                           types.String `tfsdk:"id"`
	Name                         types.String `tfsdk:"name"`
	DisplayName                  types.String `tfsdk:"display_name"`
	Enabled                      types.Bool   `tfsdk:"enabled"`
	WorkloadKind                 types.String `tfsdk:"workload_kind"`
	RoutingMode                  types.String `tfsdk:"routing_mode"`
	RouterModelType              types.String `tfsdk:"router_model_type"`
	RouterModelRef               types.String `tfsdk:"router_model_ref"`
	Routes                       types.List   `tfsdk:"routes"`
	ServerlessStartRequests      types.Int64  `tfsdk:"serverless_start_requests"`
	ServerlessStartWindowSeconds types.Int64  `tfsdk:"serverless_start_window_seconds"`
	ServingEndpointID            types.String `tfsdk:"serving_endpoint_id"`
}

type smartBalancerRouteModel struct {
	Name            types.String `tfsdk:"name"`
	Intent          types.String `tfsdk:"intent"`
	Priority        types.Int64  `tfsdk:"priority"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	IsDefault       types.Bool   `tfsdk:"is_default"`
	DestinationType types.String `tfsdk:"destination_type"`
	DestinationRef  types.String `tfsdk:"destination_ref"`
}

func smartRouteType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"name": types.StringType, "intent": types.StringType, "priority": types.Int64Type,
		"enabled": types.BoolType, "is_default": types.BoolType,
		"destination_type": types.StringType, "destination_ref": types.StringType,
	}}
}

func newSmartBalancerResource() resource.Resource { return &smartBalancerResource{} }

func (r *smartBalancerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_smart_balancer"
}

func (r *smartBalancerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Manage a QDivZero Smart Balancer and its routes.", Attributes: map[string]schema.Attribute{
		"id":                schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"name":              schema.StringAttribute{Required: true},
		"display_name":      schema.StringAttribute{Optional: true, Computed: true},
		"enabled":           schema.BoolAttribute{Optional: true, Computed: true},
		"workload_kind":     schema.StringAttribute{Optional: true, Computed: true},
		"routing_mode":      schema.StringAttribute{Optional: true, Computed: true},
		"router_model_type": schema.StringAttribute{Optional: true, Description: "Router model type; configure together with router_model_ref."},
		"router_model_ref":  schema.StringAttribute{Optional: true, Description: "Router model reference; configure together with router_model_type."},
		"routes": schema.ListNestedAttribute{Required: true, Description: "Ordered routing rules and destinations.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"name":             schema.StringAttribute{Required: true},
			"intent":           schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("")},
			"priority":         schema.Int64Attribute{Required: true},
			"enabled":          schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"is_default":       schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"destination_type": schema.StringAttribute{Required: true},
			"destination_ref":  schema.StringAttribute{Required: true},
		}}},
		"serverless_start_requests":       schema.Int64Attribute{Optional: true, Computed: true},
		"serverless_start_window_seconds": schema.Int64Attribute{Optional: true, Computed: true},
		"serving_endpoint_id":             schema.StringAttribute{Computed: true},
	}}
}

func (r *smartBalancerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*qdivzero.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected a QDivZero client, got %T.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *smartBalancerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model smartBalancerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !model.Name.IsNull() && !model.Name.IsUnknown() && strings.TrimSpace(model.Name.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid balancer name", "The name must not be empty.")
	}
	if !model.RouterModelType.IsUnknown() && !model.RouterModelRef.IsUnknown() {
		hasType := !model.RouterModelType.IsNull() && strings.TrimSpace(model.RouterModelType.ValueString()) != ""
		hasRef := !model.RouterModelRef.IsNull() && strings.TrimSpace(model.RouterModelRef.ValueString()) != ""
		if !model.RouterModelType.IsNull() && !hasType {
			resp.Diagnostics.AddAttributeError(path.Root("router_model_type"), "Invalid router model type", "The router model type must not be empty.")
		}
		if !model.RouterModelRef.IsNull() && !hasRef {
			resp.Diagnostics.AddAttributeError(path.Root("router_model_ref"), "Invalid router model reference", "The router model reference must not be empty.")
		}
		if hasType != hasRef {
			resp.Diagnostics.AddAttributeError(path.Root("router_model_ref"), "Incomplete router model", "Configure both router_model_type and router_model_ref.")
		}
	}
}

func (r *smartBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model smartBalancerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Provider not configured", "A configured QDivZero provider is required.")
		return
	}
	input, diags := model.input(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	balancer, err := r.client.CreateSmartBalancer(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create smart balancer", err.Error())
		return
	}
	if strings.TrimSpace(balancer.ID) == "" {
		resp.Diagnostics.AddError("Invalid create response", "The API did not return a balancer ID; check QDivZero before retrying.")
		return
	}
	resp.Diagnostics.Append(model.apply(ctx, balancer, false)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *smartBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model smartBalancerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.ID) {
		resp.Diagnostics.AddError("Missing provider or balancer ID", "A configured provider and balancer ID are required.")
		return
	}
	balancer, err := r.client.GetSmartBalancer(ctx, model.ID.ValueString())
	if qdivzero.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read smart balancer", err.Error())
		return
	}
	resp.Diagnostics.Append(model.apply(ctx, balancer, true)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *smartBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model smartBalancerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.ID) {
		resp.Diagnostics.AddError("Missing provider or balancer ID", "A configured provider and balancer ID are required.")
		return
	}
	input, diags := model.input(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	balancer, err := r.client.UpdateSmartBalancer(ctx, model.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update smart balancer", err.Error())
		return
	}
	if balancer.ID != model.ID.ValueString() {
		resp.Diagnostics.AddError("Invalid balancer response", "The API returned a missing or different balancer ID.")
		return
	}
	resp.Diagnostics.Append(model.apply(ctx, balancer, false)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *smartBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model smartBalancerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.ID) {
		resp.Diagnostics.AddError("Missing provider or balancer ID", "A configured provider and balancer ID are required.")
		return
	}
	if err := r.client.DeleteSmartBalancer(ctx, model.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete smart balancer", err.Error())
	}
}

func (r *smartBalancerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m smartBalancerModel) input(ctx context.Context) (qdivzero.SmartBalancerInput, diag.Diagnostics) {
	input := qdivzero.SmartBalancerInput{
		Name: m.Name.ValueString(), DisplayName: knownString(m.DisplayName), Enabled: knownBool(m.Enabled),
		WorkloadKind: knownString(m.WorkloadKind), RoutingMode: knownString(m.RoutingMode),
		ServerlessStartRequests: knownInt64(m.ServerlessStartRequests), ServerlessStartWindowSeconds: knownInt64(m.ServerlessStartWindowSeconds),
	}
	var diagnostics diag.Diagnostics
	modelType, modelRef := knownString(m.RouterModelType), knownString(m.RouterModelRef)
	if (modelType == nil) != (modelRef == nil) {
		diagnostics.AddError("Incomplete router model", "Configure both router_model_type and router_model_ref.")
	} else if modelType != nil {
		input.RouterModel = &qdivzero.BalancerDestination{Type: *modelType, Ref: *modelRef}
	}
	if !m.Routes.IsNull() && !m.Routes.IsUnknown() {
		var routes []smartBalancerRouteModel
		diagnostics.Append(m.Routes.ElementsAs(ctx, &routes, false)...)
		converted := make([]qdivzero.SmartBalancerRoute, 0, len(routes))
		for _, route := range routes {
			if route.Name.IsUnknown() || route.Intent.IsUnknown() || route.Priority.IsUnknown() || route.Enabled.IsUnknown() || route.IsDefault.IsUnknown() || route.DestinationType.IsUnknown() || route.DestinationRef.IsUnknown() || route.Name.IsNull() || route.Intent.IsNull() || route.Priority.IsNull() || route.Enabled.IsNull() || route.IsDefault.IsNull() || route.DestinationType.IsNull() || route.DestinationRef.IsNull() {
				diagnostics.AddError("Invalid route", "All route attributes must be known before creating or updating the balancer.")
				continue
			}
			converted = append(converted, qdivzero.SmartBalancerRoute{
				Name: knownString(route.Name), Intent: knownString(route.Intent), Priority: knownInt64(route.Priority),
				Enabled: knownBool(route.Enabled), IsDefault: knownBool(route.IsDefault),
				Destination: &qdivzero.BalancerDestination{Type: route.DestinationType.ValueString(), Ref: route.DestinationRef.ValueString()},
			})
		}
		input.Routes = &converted
	} else {
		diagnostics.AddError("Missing balancer routes", "Routes must be known before creating or updating a balancer.")
	}
	return input, diagnostics
}

func (m *smartBalancerModel) apply(ctx context.Context, balancer qdivzero.SmartBalancer, refresh bool) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	m.ID = types.StringValue(balancer.ID)
	setString := func(target *types.String, value *string) {
		if refresh || target.IsUnknown() {
			*target = types.StringPointerValue(value)
		}
	}
	setString(&m.Name, balancer.Name)
	setString(&m.DisplayName, balancer.DisplayName)
	setString(&m.WorkloadKind, balancer.WorkloadKind)
	setString(&m.RoutingMode, balancer.RoutingMode)
	if refresh || m.RouterModelType.IsUnknown() || m.RouterModelRef.IsUnknown() {
		var modelType, modelRef *string
		if balancer.RouterModel != nil {
			modelType, modelRef = &balancer.RouterModel.Type, &balancer.RouterModel.Ref
		}
		setString(&m.RouterModelType, modelType)
		setString(&m.RouterModelRef, modelRef)
	}
	if refresh || m.Enabled.IsUnknown() {
		m.Enabled = types.BoolPointerValue(balancer.Enabled)
	}
	if refresh || m.ServerlessStartRequests.IsUnknown() {
		m.ServerlessStartRequests = types.Int64PointerValue(balancer.ServerlessStartRequests)
	}
	if refresh || m.ServerlessStartWindowSeconds.IsUnknown() {
		m.ServerlessStartWindowSeconds = types.Int64PointerValue(balancer.ServerlessStartWindowSeconds)
	}
	m.ServingEndpointID = types.StringPointerValue(balancer.ServingEndpointID)
	if refresh {
		if balancer.Routes == nil {
			diagnostics.AddError("Incomplete balancer response", "The API did not return the balancer routes.")
			return diagnostics
		}
		routes := make([]smartBalancerRouteModel, 0, len(*balancer.Routes))
		for _, route := range *balancer.Routes {
			if route.Name == nil || route.Priority == nil || route.Destination == nil {
				diagnostics.AddError("Incomplete balancer route", "The API did not return all route fields required to manage the balancer.")
				return diagnostics
			}
			intent, enabled, isDefault := "", true, false
			if route.Intent != nil {
				intent = *route.Intent
			}
			if route.Enabled != nil {
				enabled = *route.Enabled
			}
			if route.IsDefault != nil {
				isDefault = *route.IsDefault
			}
			routes = append(routes, smartBalancerRouteModel{
				Name: types.StringPointerValue(route.Name), Intent: types.StringValue(intent), Priority: types.Int64PointerValue(route.Priority),
				Enabled: types.BoolValue(enabled), IsDefault: types.BoolValue(isDefault),
				DestinationType: types.StringValue(route.Destination.Type), DestinationRef: types.StringValue(route.Destination.Ref),
			})
		}
		value, diags := types.ListValueFrom(ctx, smartRouteType(), routes)
		diagnostics.Append(diags...)
		m.Routes = value
	}
	return diagnostics
}
