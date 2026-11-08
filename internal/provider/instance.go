package provider

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/QDivZero/qdivzero-terraform/internal/qdivzero"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &instanceResource{}
var _ resource.ResourceWithConfigure = &instanceResource{}
var _ resource.ResourceWithImportState = &instanceResource{}
var _ resource.ResourceWithValidateConfig = &instanceResource{}

type instanceResource struct{ client *qdivzero.Client }

type instanceModel struct {
	ID                                types.String  `tfsdk:"id"`
	Name                              types.String  `tfsdk:"name"`
	HuggingfaceRepoID                 types.String  `tfsdk:"huggingface_repo_id"`
	Description                       types.String  `tfsdk:"description"`
	ServingName                       types.String  `tfsdk:"serving_name"`
	RuntimePreset                     types.String  `tfsdk:"runtime_preset"`
	SchedulingMode                    types.String  `tfsdk:"scheduling_mode"`
	ServerlessEnabled                 types.Bool    `tfsdk:"serverless_enabled"`
	IdleTimeoutSeconds                types.Int64   `tfsdk:"idle_timeout_seconds"`
	State                             types.String  `tfsdk:"state"`
	CreatedAt                         types.String  `tfsdk:"created_at"`
	GPUPreferences                    types.List    `tfsdk:"gpu_preferences"`
	SmartRegion                       types.String  `tfsdk:"smart_region"`
	SmartRegions                      types.List    `tfsdk:"smart_regions"`
	SmartMaxPricePerHourEUR           types.Float64 `tfsdk:"smart_max_price_per_hour_eur"`
	SmartMinGPUClass                  types.String  `tfsdk:"smart_min_gpu_class"`
	SmartMinTotalTFLOPS               types.Float64 `tfsdk:"smart_min_total_tflops"`
	SmartAllowSpot                    types.Bool    `tfsdk:"smart_allow_spot"`
	SmartAllowCommunity               types.Bool    `tfsdk:"smart_allow_community"`
	SmartProviderPreference           types.String  `tfsdk:"smart_provider_preference"`
	SmartProviderFilterMode           types.String  `tfsdk:"smart_provider_filter_mode"`
	SmartSelectionLabel               types.String  `tfsdk:"smart_selection_label"`
	CostOptimizationEnabled           types.Bool    `tfsdk:"cost_optimization_enabled"`
	CostOptimizationMinSavingsPercent types.Float64 `tfsdk:"cost_optimization_min_savings_percent"`
	DesiredState                      types.String  `tfsdk:"desired_state"`
	WaitTimeoutSeconds                types.Int64   `tfsdk:"wait_timeout_seconds"`
	WorkloadKind                      types.String  `tfsdk:"workload_kind"`
	SupportedProtocols                types.List    `tfsdk:"supported_protocols"`
	ReplicaCount                      types.Int64   `tfsdk:"replica_count"`
	ReplicaSetID                      types.String  `tfsdk:"replica_set_id"`
}

type gpuPreferenceModel struct {
	PublicGPUID types.String `tfsdk:"public_gpu_id"`
	Count       types.Int64  `tfsdk:"count"`
}

func gpuPreferenceType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{"public_gpu_id": types.StringType, "count": types.Int64Type}}
}

func newInstanceResource() resource.Resource { return &instanceResource{} }

func (r *instanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *instanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage a QDivZero inference instance.",
		Attributes: map[string]schema.Attribute{
			"id":                   schema.StringAttribute{Computed: true, Description: "Instance ID.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":                 schema.StringAttribute{Required: true, Description: "Instance name."},
			"huggingface_repo_id":  schema.StringAttribute{Required: true, Description: "Hugging Face model repository. Changes require replacement.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"description":          schema.StringAttribute{Optional: true, Computed: true, Description: "Instance description."},
			"serving_name":         schema.StringAttribute{Optional: true, Computed: true, Description: "Serving name for the model."},
			"runtime_preset":       schema.StringAttribute{Optional: true, Computed: true, Description: "Runtime preset."},
			"scheduling_mode":      schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("smart"), Description: "Scheduling mode; defaults to smart."},
			"serverless_enabled":   schema.BoolAttribute{Optional: true, Computed: true, Description: "Whether serverless mode is enabled."},
			"idle_timeout_seconds": schema.Int64Attribute{Optional: true, Computed: true, Description: "Idle timeout in seconds."},
			"gpu_preferences": schema.ListNestedAttribute{Optional: true, Computed: true, Description: "Requested GPU types and counts.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"public_gpu_id": schema.StringAttribute{Required: true, Description: "Public GPU ID."},
				"count":         schema.Int64Attribute{Required: true, Description: "GPU count."},
			}}},
			"smart_region":                          schema.StringAttribute{Optional: true, Computed: true, Description: "Preferred Smart scheduling region."},
			"smart_regions":                         schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Description: "Allowed Smart scheduling regions."},
			"smart_max_price_per_hour_eur":          schema.Float64Attribute{Optional: true, Computed: true, Description: "Maximum Smart instance price per hour in EUR."},
			"smart_min_gpu_class":                   schema.StringAttribute{Optional: true, Computed: true, Description: "Minimum Smart GPU class."},
			"smart_min_total_tflops":                schema.Float64Attribute{Optional: true, Computed: true, Description: "Minimum total Smart compute in TFLOPS."},
			"smart_allow_spot":                      schema.BoolAttribute{Optional: true, Computed: true, Description: "Allow interruptible/spot capacity."},
			"smart_allow_community":                 schema.BoolAttribute{Optional: true, Computed: true, Description: "Allow community capacity."},
			"smart_provider_preference":             schema.StringAttribute{Optional: true, Computed: true, Description: "Preferred Smart provider."},
			"smart_provider_filter_mode":            schema.StringAttribute{Optional: true, Computed: true, Description: "Smart provider filtering mode."},
			"smart_selection_label":                 schema.StringAttribute{Optional: true, Computed: true, Description: "Smart selection label."},
			"cost_optimization_enabled":             schema.BoolAttribute{Optional: true, Computed: true, Description: "Allow replacement with cheaper compatible Smart capacity."},
			"cost_optimization_min_savings_percent": schema.Float64Attribute{Optional: true, Computed: true, Description: "Minimum percentage savings for Smart replacement."},
			"desired_state":                         schema.StringAttribute{Optional: true, Description: "Optional desired stable state: running or stopped. The provider starts or stops the instance and waits for convergence."},
			"wait_timeout_seconds":                  schema.Int64Attribute{Optional: true, Description: "Timeout for state convergence, in seconds; defaults to 1200."},
			"workload_kind":                         schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "Workload kind; changes require replacement."},
			"supported_protocols":                   schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType, Description: "Supported serving protocols."},
			"replica_count":                         schema.Int64Attribute{Optional: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}, Description: "Create-only replica count; changes require replacement."},
			"replica_set_id":                        schema.StringAttribute{Computed: true, Description: "Replica set ID, when created by the API."},
			"state":                                 schema.StringAttribute{Computed: true, Description: "Current remote instance state."},
			"created_at":                            schema.StringAttribute{Computed: true, Description: "Creation timestamp returned by the API."},
		},
	}
}

func (r *instanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *instanceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model instanceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for name, value := range map[string]types.String{"name": model.Name, "huggingface_repo_id": model.HuggingfaceRepoID} {
		if !value.IsNull() && !value.IsUnknown() && strings.TrimSpace(value.ValueString()) == "" {
			resp.Diagnostics.AddAttributeError(path.Root(name), "Invalid instance configuration", name+" must not be empty or whitespace.")
		}
	}
	if !model.DesiredState.IsNull() && !model.DesiredState.IsUnknown() && model.DesiredState.ValueString() != "running" && model.DesiredState.ValueString() != "stopped" {
		resp.Diagnostics.AddAttributeError(path.Root("desired_state"), "Invalid desired state", "The desired state must be running or stopped.")
	}
	if !model.WaitTimeoutSeconds.IsNull() && !model.WaitTimeoutSeconds.IsUnknown() && (model.WaitTimeoutSeconds.ValueInt64() < 1 || model.WaitTimeoutSeconds.ValueInt64() > int64((1<<63-1)/int64(time.Second))) {
		resp.Diagnostics.AddAttributeError(path.Root("wait_timeout_seconds"), "Invalid wait timeout", "The wait timeout must be a positive duration in seconds.")
	}
}

func (r *instanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model instanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Provider not configured", "Configure the QDivZero provider before creating an instance.")
		return
	}
	input, diagnostics := model.input(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	instance, err := r.client.CreateInstance(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create instance", err.Error())
		return
	}
	if strings.TrimSpace(instance.ID) == "" {
		resp.Diagnostics.AddError("Invalid create response", "The API did not return an instance ID; check QDivZero before retrying.")
		return
	}
	resp.Diagnostics.Append(model.apply(ctx, instance, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Preserve the created ID even if the requested state cannot be reached.
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	settled, err := r.ensureState(ctx, model, instance)
	if err != nil {
		resp.Diagnostics.AddError("Unable to reach desired instance state", err.Error())
		return
	}
	resp.Diagnostics.Append(model.apply(ctx, settled, false)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *instanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model instanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Provider not configured", "Configure the QDivZero provider before reading an instance.")
		return
	}
	if !validID(model.ID) {
		resp.Diagnostics.AddError("Missing instance ID", "An instance ID is required to read an instance.")
		return
	}
	instance, err := r.client.GetInstance(ctx, model.ID.ValueString())
	if qdivzero.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read instance", err.Error())
		return
	}
	if instance.ID != model.ID.ValueString() {
		resp.Diagnostics.AddError("Invalid read response", "The API returned a missing or different instance ID.")
		return
	}
	resp.Diagnostics.Append(model.apply(ctx, instance, true)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !model.DesiredState.IsNull() && !model.DesiredState.IsUnknown() && instance.State != nil {
		switch *instance.State {
		case "running", "stopped", "failed", "error":
			model.DesiredState = types.StringValue(*instance.State)
		case "non_bootstrap":
			model.DesiredState = types.StringValue("stopped")
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *instanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model instanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Provider not configured", "Configure the QDivZero provider before updating an instance.")
		return
	}
	if !validID(model.ID) {
		resp.Diagnostics.AddError("Missing instance ID", "An instance ID is required to update an instance.")
		return
	}
	var prior instanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input, diagnostics := model.input(ctx)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	var instance qdivzero.Instance
	var err error
	if instanceConfigurationChanged(model, prior) {
		instance, err = r.client.UpdateInstance(ctx, model.ID.ValueString(), input)
	} else {
		instance, err = r.client.GetInstance(ctx, model.ID.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to update instance", err.Error())
		return
	}
	if instance.ID != model.ID.ValueString() {
		resp.Diagnostics.AddError("Invalid update response", "The API returned a missing or different instance ID.")
		return
	}
	resp.Diagnostics.Append(model.apply(ctx, instance, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	settled, err := r.ensureState(ctx, model, instance)
	if err != nil {
		resp.Diagnostics.AddError("Unable to reach desired instance state", err.Error())
		return
	}
	resp.Diagnostics.Append(model.apply(ctx, settled, false)...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	}
}

func (r *instanceResource) ensureState(ctx context.Context, model instanceModel, instance qdivzero.Instance) (qdivzero.Instance, error) {
	if model.DesiredState.IsNull() || model.DesiredState.IsUnknown() {
		return instance, nil
	}
	timeout := int64(1200)
	if !model.WaitTimeoutSeconds.IsNull() && !model.WaitTimeoutSeconds.IsUnknown() {
		timeout = model.WaitTimeoutSeconds.ValueInt64()
	}
	if timeout < 1 || timeout > int64((1<<63-1)/int64(time.Second)) {
		return instance, fmt.Errorf("wait_timeout_seconds must be positive")
	}
	waitCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	return r.client.EnsureInstanceState(waitCtx, model.ID.ValueString(), instance, model.DesiredState.ValueString(), 5*time.Second)
}

func instanceConfigurationChanged(planned, prior instanceModel) bool {
	a, b := reflect.ValueOf(planned), reflect.ValueOf(prior)
	for i := 0; i < a.NumField(); i++ {
		name := a.Type().Field(i).Tag.Get("tfsdk")
		switch name {
		case "id", "state", "created_at", "desired_state", "wait_timeout_seconds", "replica_count", "replica_set_id":
			continue
		}
		plannedValue, ok := a.Field(i).Interface().(attr.Value)
		if !ok || plannedValue.IsUnknown() {
			continue
		}
		previousValue := b.Field(i).Interface().(attr.Value)
		if !plannedValue.Equal(previousValue) {
			return true
		}
	}
	return false
}

func (r *instanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model instanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Provider not configured", "Configure the QDivZero provider before deleting an instance.")
		return
	}
	if !validID(model.ID) {
		resp.Diagnostics.AddError("Missing instance ID", "An instance ID is required to delete an instance.")
		return
	}
	if err := r.client.DeleteInstance(ctx, model.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete instance", err.Error())
	}
}

func validID(id types.String) bool {
	return !id.IsNull() && !id.IsUnknown() && strings.TrimSpace(id.ValueString()) != ""
}

func (r *instanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m instanceModel) input(ctx context.Context) (qdivzero.InstanceInput, diag.Diagnostics) {
	input := qdivzero.InstanceInput{
		Name: m.Name.ValueString(), HuggingfaceRepoID: m.HuggingfaceRepoID.ValueString(),
		Description: knownString(m.Description), ServingName: knownString(m.ServingName),
		RuntimePreset: knownString(m.RuntimePreset), SchedulingMode: knownString(m.SchedulingMode),
		ServerlessEnabled: knownBool(m.ServerlessEnabled), IdleTimeoutSeconds: knownInt64(m.IdleTimeoutSeconds),
		SmartRegion: knownString(m.SmartRegion), SmartMaxPricePerHourEUR: knownFloat64(m.SmartMaxPricePerHourEUR),
		SmartMinGPUClass: knownString(m.SmartMinGPUClass), SmartMinTotalTFLOPS: knownFloat64(m.SmartMinTotalTFLOPS),
		SmartAllowSpot: knownBool(m.SmartAllowSpot), SmartAllowCommunity: knownBool(m.SmartAllowCommunity),
		SmartProviderPreference: knownString(m.SmartProviderPreference), SmartProviderFilterMode: knownString(m.SmartProviderFilterMode),
		SmartSelectionLabel: knownString(m.SmartSelectionLabel), CostOptimizationEnabled: knownBool(m.CostOptimizationEnabled),
		CostOptimizationMinSavingsPercent: knownFloat64(m.CostOptimizationMinSavingsPercent),
		WorkloadKind:                      knownString(m.WorkloadKind),
		ReplicaCount:                      knownInt64(m.ReplicaCount),
	}
	var diagnostics diag.Diagnostics
	if !m.SmartRegions.IsNull() && !m.SmartRegions.IsUnknown() {
		var regions []string
		diagnostics.Append(m.SmartRegions.ElementsAs(ctx, &regions, false)...)
		input.SmartRegions = &regions
	}
	if !m.SupportedProtocols.IsNull() && !m.SupportedProtocols.IsUnknown() {
		var protocols []string
		diagnostics.Append(m.SupportedProtocols.ElementsAs(ctx, &protocols, false)...)
		input.SupportedProtocols = &protocols
	}
	if !m.GPUPreferences.IsNull() && !m.GPUPreferences.IsUnknown() {
		var preferences []gpuPreferenceModel
		diagnostics.Append(m.GPUPreferences.ElementsAs(ctx, &preferences, false)...)
		converted := make([]qdivzero.GPUPreference, 0, len(preferences))
		for _, preference := range preferences {
			if preference.PublicGPUID.IsUnknown() || preference.PublicGPUID.IsNull() || preference.Count.IsUnknown() || preference.Count.IsNull() {
				diagnostics.AddError("Invalid GPU preference", "Each GPU preference must have a known GPU ID and count.")
				continue
			}
			converted = append(converted, qdivzero.GPUPreference{PublicGPUID: preference.PublicGPUID.ValueString(), Count: preference.Count.ValueInt64()})
		}
		input.GPUPreferences = &converted
	}
	return input, diagnostics
}

func knownString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueString()
	return &v
}

func knownBool(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueBool()
	return &v
}

func knownInt64(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueInt64()
	return &v
}

func knownFloat64(value types.Float64) *float64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueFloat64()
	return &v
}

func (m *instanceModel) apply(ctx context.Context, instance qdivzero.Instance, refresh bool) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	m.ID = types.StringValue(instance.ID)
	setString := func(target *types.String, value *string) {
		if refresh || target.IsUnknown() {
			*target = types.StringPointerValue(value)
		}
	}
	setString(&m.Name, instance.Name)
	setString(&m.HuggingfaceRepoID, instance.HuggingfaceRepoID)
	setString(&m.Description, instance.Description)
	setString(&m.ServingName, instance.ServingName)
	setString(&m.RuntimePreset, instance.RuntimePreset)
	setString(&m.SchedulingMode, instance.SchedulingMode)
	setString(&m.SmartRegion, instance.SmartRegion)
	setString(&m.SmartMinGPUClass, instance.SmartMinGPUClass)
	setString(&m.SmartProviderPreference, instance.SmartProviderPreference)
	setString(&m.SmartProviderFilterMode, instance.SmartProviderFilterMode)
	setString(&m.SmartSelectionLabel, instance.SmartSelectionLabel)
	setString(&m.WorkloadKind, instance.WorkloadKind)
	setBool := func(target *types.Bool, value *bool) {
		if refresh || target.IsUnknown() {
			*target = types.BoolPointerValue(value)
		}
	}
	setFloat := func(target *types.Float64, value *float64) {
		if refresh || target.IsUnknown() {
			*target = types.Float64PointerValue(value)
		}
	}
	setBool(&m.SmartAllowSpot, instance.SmartAllowSpot)
	setBool(&m.SmartAllowCommunity, instance.SmartAllowCommunity)
	setBool(&m.CostOptimizationEnabled, instance.CostOptimizationEnabled)
	setFloat(&m.SmartMaxPricePerHourEUR, instance.SmartMaxPricePerHourEUR)
	setFloat(&m.SmartMinTotalTFLOPS, instance.SmartMinTotalTFLOPS)
	setFloat(&m.CostOptimizationMinSavingsPercent, instance.CostOptimizationMinSavingsPercent)
	if refresh || m.ServerlessEnabled.IsUnknown() {
		m.ServerlessEnabled = types.BoolPointerValue(instance.ServerlessEnabled)
	}
	if refresh || m.IdleTimeoutSeconds.IsUnknown() {
		m.IdleTimeoutSeconds = types.Int64PointerValue(instance.IdleTimeoutSeconds)
	}
	if refresh || m.SmartRegions.IsUnknown() {
		m.SmartRegions = types.ListNull(types.StringType)
		if instance.SmartRegions != nil {
			value, diags := types.ListValueFrom(ctx, types.StringType, *instance.SmartRegions)
			diagnostics.Append(diags...)
			m.SmartRegions = value
		}
	}
	if refresh || m.SupportedProtocols.IsUnknown() {
		m.SupportedProtocols = types.ListNull(types.StringType)
		if instance.SupportedProtocols != nil {
			value, diags := types.ListValueFrom(ctx, types.StringType, *instance.SupportedProtocols)
			diagnostics.Append(diags...)
			m.SupportedProtocols = value
		}
	}
	if refresh || m.GPUPreferences.IsUnknown() {
		m.GPUPreferences = types.ListNull(gpuPreferenceType())
		if instance.GPUPreferences != nil {
			preferences := make([]gpuPreferenceModel, 0, len(*instance.GPUPreferences))
			for _, preference := range *instance.GPUPreferences {
				preferences = append(preferences, gpuPreferenceModel{PublicGPUID: types.StringValue(preference.PublicGPUID), Count: types.Int64Value(preference.Count)})
			}
			value, diags := types.ListValueFrom(ctx, gpuPreferenceType(), preferences)
			diagnostics.Append(diags...)
			m.GPUPreferences = value
		}
	}
	m.State = types.StringPointerValue(instance.State)
	if refresh || m.ReplicaSetID.IsUnknown() {
		m.ReplicaSetID = types.StringPointerValue(instance.ReplicaSetID)
	}
	if refresh || m.CreatedAt.IsUnknown() {
		m.CreatedAt = types.StringPointerValue(instance.CreatedAt)
	}
	return diagnostics
}
