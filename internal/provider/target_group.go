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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &targetGroupResource{}
var _ resource.ResourceWithConfigure = &targetGroupResource{}
var _ resource.ResourceWithImportState = &targetGroupResource{}
var _ resource.ResourceWithValidateConfig = &targetGroupResource{}

type targetGroupResource struct{ client *qdivzero.Client }

type targetGroupModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	DisplayName     types.String `tfsdk:"display_name"`
	Description     types.String `tfsdk:"description"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	WorkloadKind    types.String `tfsdk:"workload_kind"`
	SelectionPolicy types.String `tfsdk:"selection_policy"`
	Members         types.List   `tfsdk:"members"`
}

type targetGroupMemberModel struct {
	InstanceID types.String `tfsdk:"instance_id"`
	Weight     types.Int64  `tfsdk:"weight"`
	Priority   types.Int64  `tfsdk:"priority"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

func targetMemberType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{"instance_id": types.StringType, "weight": types.Int64Type, "priority": types.Int64Type, "enabled": types.BoolType}}
}

func newTargetGroupResource() resource.Resource { return &targetGroupResource{} }

func (r *targetGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target_group"
}

func (r *targetGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Manage a QDivZero target group of inference instances.", Attributes: map[string]schema.Attribute{
		"id":               schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"name":             schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"display_name":     schema.StringAttribute{Optional: true, Computed: true},
		"description":      schema.StringAttribute{Optional: true, Computed: true},
		"enabled":          schema.BoolAttribute{Optional: true, Computed: true},
		"workload_kind":    schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"selection_policy": schema.StringAttribute{Optional: true, Computed: true, Description: "Selection policy such as priority, weighted, or round_robin."},
		"members": schema.ListNestedAttribute{Required: true, Description: "Instances and their priority and weight.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"instance_id": schema.StringAttribute{Required: true},
			"weight":      schema.Int64Attribute{Required: true},
			"priority":    schema.Int64Attribute{Required: true},
			"enabled":     schema.BoolAttribute{Required: true},
		}}},
	}}
}

func (r *targetGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *targetGroupResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model targetGroupModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !model.Name.IsNull() && !model.Name.IsUnknown() && strings.TrimSpace(model.Name.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid target group name", "The name must not be empty.")
	}
}

func (r *targetGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model targetGroupModel
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
	group, err := r.client.CreateTargetGroup(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create target group", err.Error())
		return
	}
	if strings.TrimSpace(group.ID) == "" {
		resp.Diagnostics.AddError("Invalid create response", "The API did not return a group ID; check QDivZero before retrying.")
		return
	}
	resp.Diagnostics.Append(model.apply(ctx, group, false)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *targetGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model targetGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.ID) {
		resp.Diagnostics.AddError("Missing provider or group ID", "A configured provider and target group ID are required.")
		return
	}
	group, err := r.client.GetTargetGroup(ctx, model.ID.ValueString())
	if qdivzero.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read target group", err.Error())
		return
	}
	if group.ID != model.ID.ValueString() {
		resp.Diagnostics.AddError("Invalid target group response", "The API returned a missing or different target group ID.")
		return
	}
	resp.Diagnostics.Append(model.apply(ctx, group, true)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *targetGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model targetGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.ID) {
		resp.Diagnostics.AddError("Missing provider or group ID", "A configured provider and target group ID are required.")
		return
	}
	input, diags := model.input(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	group, err := r.client.UpdateTargetGroup(ctx, model.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update target group", err.Error())
		return
	}
	if group.ID != model.ID.ValueString() {
		resp.Diagnostics.AddError("Invalid target group response", "The API returned a missing or different target group ID.")
		return
	}
	resp.Diagnostics.Append(model.apply(ctx, group, false)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *targetGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model targetGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.ID) {
		resp.Diagnostics.AddError("Missing provider or group ID", "A configured provider and target group ID are required.")
		return
	}
	if err := r.client.DeleteTargetGroup(ctx, model.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete target group", err.Error())
	}
}

func (r *targetGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m targetGroupModel) input(ctx context.Context) (qdivzero.TargetGroupInput, diag.Diagnostics) {
	input := qdivzero.TargetGroupInput{Name: m.Name.ValueString(), DisplayName: knownString(m.DisplayName), Description: knownString(m.Description), Enabled: knownBool(m.Enabled), WorkloadKind: knownString(m.WorkloadKind), SelectionPolicy: knownString(m.SelectionPolicy)}
	var diagnostics diag.Diagnostics
	if m.Members.IsNull() || m.Members.IsUnknown() {
		diagnostics.AddError("Missing target group members", "Members must be known before creating or updating a target group.")
		return input, diagnostics
	}
	var members []targetGroupMemberModel
	diagnostics.Append(m.Members.ElementsAs(ctx, &members, false)...)
	converted := make([]qdivzero.TargetGroupMember, 0, len(members))
	for _, member := range members {
		if member.InstanceID.IsNull() || member.InstanceID.IsUnknown() || member.Weight.IsNull() || member.Weight.IsUnknown() || member.Priority.IsNull() || member.Priority.IsUnknown() || member.Enabled.IsNull() || member.Enabled.IsUnknown() {
			diagnostics.AddError("Invalid target group member", "Every member must have a known instance ID, weight, priority, and enabled flag.")
			continue
		}
		converted = append(converted, qdivzero.TargetGroupMember{InstanceID: member.InstanceID.ValueString(), Weight: member.Weight.ValueInt64(), Priority: member.Priority.ValueInt64(), Enabled: member.Enabled.ValueBool()})
	}
	input.Members = &converted
	return input, diagnostics
}

func (m *targetGroupModel) apply(ctx context.Context, group qdivzero.TargetGroup, refresh bool) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	m.ID = types.StringValue(group.ID)
	setString := func(target *types.String, value *string) {
		if refresh || target.IsUnknown() {
			*target = types.StringPointerValue(value)
		}
	}
	setString(&m.Name, group.Name)
	setString(&m.DisplayName, group.DisplayName)
	setString(&m.Description, group.Description)
	setString(&m.WorkloadKind, group.WorkloadKind)
	setString(&m.SelectionPolicy, group.SelectionPolicy)
	if refresh || m.Enabled.IsUnknown() {
		m.Enabled = types.BoolPointerValue(group.Enabled)
	}
	if refresh {
		if group.Members == nil {
			diagnostics.AddError("Incomplete target group response", "The API did not return the target group members.")
			return diagnostics
		}
		members := make([]targetGroupMemberModel, 0, len(*group.Members))
		for _, member := range *group.Members {
			members = append(members, targetGroupMemberModel{InstanceID: types.StringValue(member.InstanceID), Weight: types.Int64Value(member.Weight), Priority: types.Int64Value(member.Priority), Enabled: types.BoolValue(member.Enabled)})
		}
		value, diags := types.ListValueFrom(ctx, targetMemberType(), members)
		diagnostics.Append(diags...)
		m.Members = value
	}
	return diagnostics
}
