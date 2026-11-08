package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/QDivZero/qdivzero-terraform/internal/qdivzero"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &scheduleRuleResource{}
var _ resource.ResourceWithConfigure = &scheduleRuleResource{}
var _ resource.ResourceWithImportState = &scheduleRuleResource{}
var _ resource.ResourceWithValidateConfig = &scheduleRuleResource{}

type scheduleRuleResource struct{ client *qdivzero.Client }

type scheduleRuleModel struct {
	ID              types.String `tfsdk:"id"`
	InstanceID      types.String `tfsdk:"instance_id"`
	Action          types.String `tfsdk:"action"`
	Cron            types.String `tfsdk:"cron"`
	Timezone        types.String `tfsdk:"timezone"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	DesiredReplicas types.Int64  `tfsdk:"desired_replicas"`
	ReplicaSetID    types.String `tfsdk:"replica_set_id"`
	NextRunAt       types.String `tfsdk:"next_run_at"`
	LastRunAt       types.String `tfsdk:"last_run_at"`
}

func newScheduleRuleResource() resource.Resource { return &scheduleRuleResource{} }

func (r *scheduleRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_schedule"
}

func (r *scheduleRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Manage a schedule rule for a QDivZero instance.", Attributes: map[string]schema.Attribute{
		"id":               schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"instance_id":      schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"action":           schema.StringAttribute{Required: true, Description: "Schedule action: start, stop, or scale."},
		"cron":             schema.StringAttribute{Required: true, Description: "Cron expression."},
		"timezone":         schema.StringAttribute{Optional: true, Computed: true, Description: "IANA timezone."},
		"enabled":          schema.BoolAttribute{Optional: true, Computed: true},
		"desired_replicas": schema.Int64Attribute{Optional: true, Computed: true},
		"replica_set_id":   schema.StringAttribute{Optional: true, Computed: true},
		"next_run_at":      schema.StringAttribute{Computed: true},
		"last_run_at":      schema.StringAttribute{Computed: true},
	}}
}

func (r *scheduleRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *scheduleRuleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model scheduleRuleModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for name, value := range map[string]types.String{"instance_id": model.InstanceID, "action": model.Action, "cron": model.Cron} {
		if !value.IsNull() && !value.IsUnknown() && strings.TrimSpace(value.ValueString()) == "" {
			resp.Diagnostics.AddAttributeError(path.Root(name), "Invalid schedule configuration", name+" must not be empty.")
		}
	}
	if !model.Action.IsNull() && !model.Action.IsUnknown() {
		switch model.Action.ValueString() {
		case "start", "stop", "scale":
		default:
			resp.Diagnostics.AddAttributeError(path.Root("action"), "Invalid schedule action", "Action must be start, stop, or scale.")
		}
	}
}

func (r *scheduleRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model scheduleRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.InstanceID) {
		resp.Diagnostics.AddError("Missing provider or instance", "A configured provider and instance ID are required.")
		return
	}
	rule, err := r.client.CreateScheduleRule(ctx, model.InstanceID.ValueString(), model.input())
	if err != nil {
		resp.Diagnostics.AddError("Unable to create schedule", err.Error())
		return
	}
	if strings.TrimSpace(rule.ID) == "" {
		resp.Diagnostics.AddError("Invalid schedule response", "The API did not return a schedule rule ID.")
		return
	}
	model.apply(rule, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *scheduleRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model scheduleRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.InstanceID) || !validID(model.ID) {
		resp.Diagnostics.AddError("Missing provider or schedule ID", "A configured provider, instance ID, and schedule rule ID are required.")
		return
	}
	rule, err := r.client.GetScheduleRule(ctx, model.InstanceID.ValueString(), model.ID.ValueString())
	if qdivzero.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read schedule", err.Error())
		return
	}
	model.apply(rule, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *scheduleRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model scheduleRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.InstanceID) || !validID(model.ID) {
		resp.Diagnostics.AddError("Missing provider or schedule ID", "A configured provider, instance ID, and schedule rule ID are required.")
		return
	}
	rule, err := r.client.UpdateScheduleRule(ctx, model.InstanceID.ValueString(), model.ID.ValueString(), model.input())
	if err != nil {
		resp.Diagnostics.AddError("Unable to update schedule", err.Error())
		return
	}
	if rule.ID != model.ID.ValueString() {
		resp.Diagnostics.AddError("Invalid schedule response", "The API returned a missing or different schedule rule ID.")
		return
	}
	model.apply(rule, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *scheduleRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model scheduleRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.InstanceID) || !validID(model.ID) {
		resp.Diagnostics.AddError("Missing provider or schedule ID", "A configured provider, instance ID, and schedule rule ID are required.")
		return
	}
	if err := r.client.DeleteScheduleRule(ctx, model.InstanceID.ValueString(), model.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete schedule", err.Error())
	}
}

func (r *scheduleRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Use instance_id/rule_id to import a schedule.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (m scheduleRuleModel) input() qdivzero.ScheduleRuleInput {
	return qdivzero.ScheduleRuleInput{Action: m.Action.ValueString(), Cron: m.Cron.ValueString(), Timezone: knownString(m.Timezone), Enabled: knownBool(m.Enabled), DesiredReplicas: knownInt64(m.DesiredReplicas), ReplicaSetID: knownString(m.ReplicaSetID)}
}

func (m *scheduleRuleModel) apply(rule qdivzero.ScheduleRule, refresh bool) {
	m.ID = types.StringValue(rule.ID)
	setString := func(target *types.String, value *string) {
		if refresh || target.IsUnknown() {
			*target = types.StringPointerValue(value)
		}
	}
	setString(&m.Action, rule.Action)
	setString(&m.Cron, rule.Cron)
	setString(&m.Timezone, rule.Timezone)
	setString(&m.ReplicaSetID, rule.ReplicaSetID)
	if refresh || m.Enabled.IsUnknown() {
		m.Enabled = types.BoolPointerValue(rule.Enabled)
	}
	if refresh || m.DesiredReplicas.IsUnknown() {
		m.DesiredReplicas = types.Int64PointerValue(rule.DesiredReplicas)
	}
	m.NextRunAt = types.StringPointerValue(rule.NextRunAt)
	m.LastRunAt = types.StringPointerValue(rule.LastRunAt)
}
