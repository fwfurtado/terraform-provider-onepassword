package onepasswordprovider

import (
	"context"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnePasswordDatabaseModel represents database items.
type OnePasswordDatabaseModel struct {
	SharedItemModel
	Type              types.String                   `tfsdk:"type"`
	Server            types.String                   `tfsdk:"server"`
	Port              types.Int64                    `tfsdk:"port"`
	Database          types.String                   `tfsdk:"database"`
	Username          types.String                   `tfsdk:"username"`
	Password          OnePasswordSharedPasswordModel `tfsdk:"password"`
	SID               types.String                   `tfsdk:"sid"`
	Alias             types.String                   `tfsdk:"alias"`
	ConnectionOptions types.String                   `tfsdk:"connection_options"`
}

// OnePasswordDatabase manages database items.
type OnePasswordDatabase struct {
	BaseItemResource
}

var (
	_ resource.ResourceWithConfigure = &OnePasswordDatabase{}
)

// Metadata sets the resource type name.
func (r *OnePasswordDatabase) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

// Schema defines the schema for database items.
func (r *OnePasswordDatabase) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := sharedItemAttributes()
	attributes["type"] = schema.StringAttribute{
		MarkdownDescription: "Database type.",
		Optional:            true,
	}
	attributes["server"] = schema.StringAttribute{
		MarkdownDescription: "Database server.",
		Optional:            true,
	}
	attributes["port"] = schema.Int64Attribute{
		MarkdownDescription: "Database port.",
		Optional:            true,
	}
	attributes["database"] = schema.StringAttribute{
		MarkdownDescription: "Database name.",
		Optional:            true,
	}
	attributes["username"] = schema.StringAttribute{
		MarkdownDescription: "Database username.",
		Optional:            true,
	}
	attributes["sid"] = schema.StringAttribute{
		MarkdownDescription: "Database SID.",
		Optional:            true,
	}
	attributes["alias"] = schema.StringAttribute{
		MarkdownDescription: "Database alias.",
		Optional:            true,
	}
	attributes["connection_options"] = schema.StringAttribute{
		MarkdownDescription: "Connection options.",
		Optional:            true,
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage 1Password database items.",
		Attributes:          attributes,
		Blocks: map[string]schema.Block{
			"password": passwordBlockSchema("Password recipe for the database."),
		},
	}
}

// Create creates a database item.
func (r *OnePasswordDatabase) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OnePasswordDatabaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := r.client.GetVault(ctx, plan.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get vault", err.Error())
		return
	}

	password, _, err := generatePasswordFromShared(ctx, r.client, &plan.Password, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate password", err.Error())
		return
	}

	inputs := []FieldInput{}
	addStringField(&inputs, "type", onepassword.ItemFieldTypeText, plan.Type)
	addStringField(&inputs, "server", onepassword.ItemFieldTypeText, plan.Server)
	addIntField(&inputs, "port", onepassword.ItemFieldTypeText, plan.Port)
	addStringField(&inputs, "database", onepassword.ItemFieldTypeText, plan.Database)
	addStringField(&inputs, "username", onepassword.ItemFieldTypeText, plan.Username)
	addConcealedField(&inputs, "password", types.StringValue(password))
	addStringField(&inputs, "sid", onepassword.ItemFieldTypeText, plan.SID)
	addStringField(&inputs, "alias", onepassword.ItemFieldTypeText, plan.Alias)
	addStringField(&inputs, "connection_options", onepassword.ItemFieldTypeText, plan.ConnectionOptions)
	extraFields := buildFieldsFromInputs(inputs, nil)

	params, err := buildItemCreateParamsFromShared(onepassword.ItemCategoryDatabase, vaultID, plan.SharedItemModel, extraFields, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build item parameters", err.Error())
		return
	}

	item, err := r.client.CreateItem(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create item", err.Error())
		return
	}

	plan.ID = types.StringValue(item.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes state for a database item.
func (r *OnePasswordDatabase) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OnePasswordDatabaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := r.client.GetVault(ctx, state.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get vault", err.Error())
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		resp.Diagnostics.AddError("Missing item ID", "The state does not contain an item ID.")
		return
	}

	if _, err := r.client.GetItem(ctx, vaultID, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to read item", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates a database item.
func (r *OnePasswordDatabase) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OnePasswordDatabaseModel
	var state OnePasswordDatabaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := r.client.GetVault(ctx, plan.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get vault", err.Error())
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		resp.Diagnostics.AddError("Missing item ID", "The state does not contain an item ID.")
		return
	}

	existing, err := r.client.GetItem(ctx, vaultID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to load existing item", err.Error())
		return
	}

	password, _, err := generatePasswordFromShared(ctx, r.client, &plan.Password, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate password", err.Error())
		return
	}

	inputs := []FieldInput{}
	addStringField(&inputs, "type", onepassword.ItemFieldTypeText, plan.Type)
	addStringField(&inputs, "server", onepassword.ItemFieldTypeText, plan.Server)
	addIntField(&inputs, "port", onepassword.ItemFieldTypeText, plan.Port)
	addStringField(&inputs, "database", onepassword.ItemFieldTypeText, plan.Database)
	addStringField(&inputs, "username", onepassword.ItemFieldTypeText, plan.Username)
	addConcealedField(&inputs, "password", types.StringValue(password))
	addStringField(&inputs, "sid", onepassword.ItemFieldTypeText, plan.SID)
	addStringField(&inputs, "alias", onepassword.ItemFieldTypeText, plan.Alias)
	addStringField(&inputs, "connection_options", onepassword.ItemFieldTypeText, plan.ConnectionOptions)
	extraFields := buildFieldsFromInputs(inputs, existing)

	item, err := buildItemForUpdateFromShared(onepassword.ItemCategoryDatabase, vaultID, plan.SharedItemModel, existing, extraFields, nil, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build item update", err.Error())
		return
	}

	updatedItem, err := r.client.UpdateItem(ctx, item)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update item", err.Error())
		return
	}

	plan.ID = types.StringValue(updatedItem.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a database item.
func (r *OnePasswordDatabase) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OnePasswordDatabaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ID.IsNull() || state.ID.IsUnknown() {
		resp.Diagnostics.AddError("Missing item ID", "The state does not contain an item ID.")
		return
	}

	vaultID, err := r.client.GetVault(ctx, state.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get vault", err.Error())
		return
	}

	if err := r.client.DeleteItem(ctx, vaultID, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete item", err.Error())
		return
	}
}
