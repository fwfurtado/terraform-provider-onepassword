package onepasswordprovider

import (
	"context"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnePasswordPasswordItemModel represents the password resource.
type OnePasswordPasswordItemModel struct {
	SharedItemModel
	Password OnePasswordSharedPasswordModel        `tfsdk:"password"`
	Websites map[string]OnePasswordWebsiteMapModel `tfsdk:"websites"`
}

// OnePasswordPasswordItem manages password items.
type OnePasswordPasswordItem struct {
	BaseItemResource
}

var (
	_ resource.ResourceWithConfigure = &OnePasswordPasswordItem{}
)

// Metadata sets the resource type name.
func (r *OnePasswordPasswordItem) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_password"
}

// Schema defines the schema for the password resource.
func (r *OnePasswordPasswordItem) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := sharedItemAttributes()
	attributes["websites"] = schema.MapNestedAttribute{
		MarkdownDescription: "Website entries keyed by name.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"url": schema.StringAttribute{
					MarkdownDescription: "Website URL.",
					Required:            true,
				},
				"autofill_behavior": schema.StringAttribute{
					MarkdownDescription: "Website autofill behavior.",
					Optional:            true,
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage 1Password password items.",
		Attributes:          attributes,
		Blocks: map[string]schema.Block{
			"password": passwordBlockSchema("Password recipe for this item."),
		},
	}
}

// Create creates a password item.
func (r *OnePasswordPasswordItem) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OnePasswordPasswordItemModel
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
	addConcealedField(&inputs, "password", types.StringValue(password))
	extraFields := buildFieldsFromInputs(inputs, nil)

	websites, err := mapWebsitesFromMap(plan.Websites)
	if err != nil {
		resp.Diagnostics.AddError("Failed to map websites", err.Error())
		return
	}

	params, err := buildItemCreateParamsFromShared(onepassword.ItemCategoryPassword, vaultID, plan.SharedItemModel, extraFields, websites, nil)
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

// Read refreshes state for a password item.
func (r *OnePasswordPasswordItem) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OnePasswordPasswordItemModel
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

// Update updates a password item.
func (r *OnePasswordPasswordItem) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OnePasswordPasswordItemModel
	var state OnePasswordPasswordItemModel
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
	addConcealedField(&inputs, "password", types.StringValue(password))
	extraFields := buildFieldsFromInputs(inputs, existing)

	websites, err := mapWebsitesFromMap(plan.Websites)
	if err != nil {
		resp.Diagnostics.AddError("Failed to map websites", err.Error())
		return
	}

	item, err := buildItemForUpdateFromShared(onepassword.ItemCategoryPassword, vaultID, plan.SharedItemModel, existing, extraFields, websites, nil)
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

// Delete removes a password item.
func (r *OnePasswordPasswordItem) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OnePasswordPasswordItemModel
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
