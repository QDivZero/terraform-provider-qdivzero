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

var _ resource.Resource = &vectorDatabaseResource{}
var _ resource.ResourceWithConfigure = &vectorDatabaseResource{}
var _ resource.ResourceWithImportState = &vectorDatabaseResource{}
var _ resource.ResourceWithValidateConfig = &vectorDatabaseResource{}

type vectorDatabaseResource struct{ client *qdivzero.Client }

type vectorDatabaseModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	EmbeddingServingName types.String `tfsdk:"embedding_serving_name"`
	Slug                 types.String `tfsdk:"slug"`
	Description          types.String `tfsdk:"description"`
	Status               types.String `tfsdk:"status"`
	CreatedAt            types.String `tfsdk:"created_at"`
}

func newVectorDatabaseResource() resource.Resource { return &vectorDatabaseResource{} }

func (r *vectorDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vector_database"
}

func (r *vectorDatabaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "Manage a QDivZero flexible vector database.", Attributes: map[string]schema.Attribute{
		"id":                     schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"name":                   schema.StringAttribute{Required: true},
		"embedding_serving_name": schema.StringAttribute{Required: true, Description: "Serving name of a compatible embeddings model."},
		"slug":                   schema.StringAttribute{Optional: true, Computed: true},
		"description":            schema.StringAttribute{Optional: true, Computed: true},
		"status":                 schema.StringAttribute{Computed: true},
		"created_at":             schema.StringAttribute{Computed: true},
	}}
}

func (r *vectorDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *vectorDatabaseResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model vectorDatabaseModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for name, value := range map[string]types.String{"name": model.Name, "embedding_serving_name": model.EmbeddingServingName} {
		if !value.IsNull() && !value.IsUnknown() && strings.TrimSpace(value.ValueString()) == "" {
			resp.Diagnostics.AddAttributeError(path.Root(name), "Invalid vector database configuration", name+" must not be empty.")
		}
	}
}

func (r *vectorDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model vectorDatabaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil {
		resp.Diagnostics.AddError("Provider not configured", "A configured QDivZero provider is required.")
		return
	}
	database, err := r.client.CreateVectorDatabase(ctx, model.input())
	if err != nil {
		resp.Diagnostics.AddError("Unable to create vector database", err.Error())
		return
	}
	if strings.TrimSpace(database.ID) == "" {
		resp.Diagnostics.AddError("Invalid create response", "The API did not return a database ID; check QDivZero before retrying.")
		return
	}
	model.apply(database, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *vectorDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model vectorDatabaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.ID) {
		resp.Diagnostics.AddError("Missing provider or database ID", "A configured provider and database ID are required.")
		return
	}
	database, err := r.client.GetVectorDatabase(ctx, model.ID.ValueString())
	if qdivzero.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read vector database", err.Error())
		return
	}
	if database.ID != model.ID.ValueString() {
		resp.Diagnostics.AddError("Invalid database response", "The API returned a missing or different database ID.")
		return
	}
	model.apply(database, true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *vectorDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model vectorDatabaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.ID) {
		resp.Diagnostics.AddError("Missing provider or database ID", "A configured provider and database ID are required.")
		return
	}
	database, err := r.client.UpdateVectorDatabase(ctx, model.ID.ValueString(), model.input())
	if err != nil {
		resp.Diagnostics.AddError("Unable to update vector database", err.Error())
		return
	}
	if database.ID != model.ID.ValueString() {
		resp.Diagnostics.AddError("Invalid database response", "The API returned a missing or different database ID.")
		return
	}
	model.apply(database, false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *vectorDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model vectorDatabaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !validID(model.ID) {
		resp.Diagnostics.AddError("Missing provider or database ID", "A configured provider and database ID are required.")
		return
	}
	if err := r.client.DeleteVectorDatabase(ctx, model.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to delete vector database", err.Error())
	}
}

func (r *vectorDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (m vectorDatabaseModel) input() qdivzero.VectorDatabaseInput {
	return qdivzero.VectorDatabaseInput{Name: m.Name.ValueString(), EmbeddingServingName: m.EmbeddingServingName.ValueString(), Slug: knownString(m.Slug), Description: knownString(m.Description)}
}

func (m *vectorDatabaseModel) apply(database qdivzero.VectorDatabase, refresh bool) {
	m.ID = types.StringValue(database.ID)
	set := func(target *types.String, value *string) {
		if refresh || target.IsUnknown() {
			*target = types.StringPointerValue(value)
		}
	}
	set(&m.Name, database.Name)
	set(&m.EmbeddingServingName, database.EmbeddingServingName)
	set(&m.Slug, database.Slug)
	set(&m.Description, database.Description)
	m.Status = types.StringPointerValue(database.Status)
	if refresh || m.CreatedAt.IsUnknown() {
		m.CreatedAt = types.StringPointerValue(database.CreatedAt)
	}
}
