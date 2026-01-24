package onepasswordprovider

import (
	"context"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnePasswordAPICredentialsModel represents API credentials items.
type OnePasswordAPICredentialsModel struct {
	SharedItemModel
	Username   types.String                   `tfsdk:"username"`
	Credential OnePasswordSharedPasswordModel `tfsdk:"credential"`
	Type       types.String                   `tfsdk:"type"`
	Filename   types.String                   `tfsdk:"filename"`
	ValidFrom  types.String                   `tfsdk:"valid_from"`
	Expires    types.String                   `tfsdk:"expires"`
	Hostname   types.String                   `tfsdk:"hostname"`
}

// OnePasswordAPICredentials manages API credentials items.
type OnePasswordAPICredentials struct {
	BaseItemResource
}

var (
	_ resource.ResourceWithConfigure = &OnePasswordAPICredentials{}
)

// Metadata sets the resource type name.
func (r *OnePasswordAPICredentials) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_credentials"
}

// Schema defines the schema for API credentials.
func (r *OnePasswordAPICredentials) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := sharedItemAttributes()
	attributes["username"] = schema.StringAttribute{
		MarkdownDescription: "API username.",
		Optional:            true,
	}
	attributes["type"] = schema.StringAttribute{
		MarkdownDescription: "Credential type.",
		Optional:            true,
	}
	attributes["filename"] = schema.StringAttribute{
		MarkdownDescription: "Credential filename.",
		Optional:            true,
	}
	attributes["valid_from"] = schema.StringAttribute{
		MarkdownDescription: "Valid from date.",
		Optional:            true,
	}
	attributes["expires"] = schema.StringAttribute{
		MarkdownDescription: "Expiration date.",
		Optional:            true,
	}
	attributes["hostname"] = schema.StringAttribute{
		MarkdownDescription: "Hostname.",
		Optional:            true,
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage 1Password API credentials items.",
		Attributes:          attributes,
		Blocks: map[string]schema.Block{
			"credential": passwordBlockSchema("Credential recipe for API access."),
		},
	}
}

// Create creates an API credentials item.
func (r *OnePasswordAPICredentials) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OnePasswordAPICredentialsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := r.client.GetVault(ctx, plan.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get vault", err.Error())
		return
	}

	credential, _, err := generatePasswordFromShared(ctx, r.client, &plan.Credential, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate credential", err.Error())
		return
	}

	inputs := []FieldInput{}
	addStringField(&inputs, "username", onepassword.ItemFieldTypeText, plan.Username)
	addConcealedField(&inputs, "credential", types.StringValue(credential))
	addStringField(&inputs, "type", onepassword.ItemFieldTypeText, plan.Type)
	addStringField(&inputs, "filename", onepassword.ItemFieldTypeText, plan.Filename)
	addStringField(&inputs, "valid_from", onepassword.ItemFieldTypeText, plan.ValidFrom)
	addStringField(&inputs, "expires", onepassword.ItemFieldTypeText, plan.Expires)
	addStringField(&inputs, "hostname", onepassword.ItemFieldTypeText, plan.Hostname)
	extraFields := buildFieldsFromInputs(inputs, nil)

	params, err := buildItemCreateParamsFromShared(onepassword.ItemCategoryAPICredentials, vaultID, plan.SharedItemModel, extraFields, nil, nil)
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

// Read refreshes state for API credentials.
func (r *OnePasswordAPICredentials) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OnePasswordAPICredentialsModel
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

// Update updates an API credentials item.
func (r *OnePasswordAPICredentials) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OnePasswordAPICredentialsModel
	var state OnePasswordAPICredentialsModel
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

	credential, _, err := generatePasswordFromShared(ctx, r.client, &plan.Credential, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate credential", err.Error())
		return
	}

	inputs := []FieldInput{}
	addStringField(&inputs, "username", onepassword.ItemFieldTypeText, plan.Username)
	addConcealedField(&inputs, "credential", types.StringValue(credential))
	addStringField(&inputs, "type", onepassword.ItemFieldTypeText, plan.Type)
	addStringField(&inputs, "filename", onepassword.ItemFieldTypeText, plan.Filename)
	addStringField(&inputs, "valid_from", onepassword.ItemFieldTypeText, plan.ValidFrom)
	addStringField(&inputs, "expires", onepassword.ItemFieldTypeText, plan.Expires)
	addStringField(&inputs, "hostname", onepassword.ItemFieldTypeText, plan.Hostname)
	extraFields := buildFieldsFromInputs(inputs, existing)

	item, err := buildItemForUpdateFromShared(onepassword.ItemCategoryAPICredentials, vaultID, plan.SharedItemModel, existing, extraFields, nil, nil)
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

// Delete removes an API credentials item.
func (r *OnePasswordAPICredentials) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OnePasswordAPICredentialsModel
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
