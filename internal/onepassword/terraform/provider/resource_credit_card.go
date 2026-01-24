package onepasswordprovider

import (
	"context"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnePasswordCreditCardModel represents the credit card resource.
type OnePasswordCreditCardModel struct {
	SharedItemModel
	Cardholder types.String `tfsdk:"cardholder"`
	Number     types.String `tfsdk:"number"`
	Type       types.String `tfsdk:"type"`
	Expiry     types.String `tfsdk:"expiry"`
}

// OnePasswordCreditCard manages credit card items.
type OnePasswordCreditCard struct {
	BaseItemResource
}

var (
	_ resource.ResourceWithConfigure = &OnePasswordCreditCard{}
)

// Metadata sets the resource type name.
func (r *OnePasswordCreditCard) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_credit_card"
}

// Schema defines the schema for the credit card resource.
func (r *OnePasswordCreditCard) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := sharedItemAttributes()
	attributes["cardholder"] = schema.StringAttribute{
		MarkdownDescription: "Cardholder name.",
		Required:            true,
	}
	attributes["number"] = schema.StringAttribute{
		MarkdownDescription: "Card number.",
		Required:            true,
		Sensitive:           true,
		WriteOnly:           true,
	}
	attributes["type"] = schema.StringAttribute{
		MarkdownDescription: "Card type.",
		Required:            true,
	}
	attributes["expiry"] = schema.StringAttribute{
		MarkdownDescription: "Expiry date (MM/YYYY).",
		Required:            true,
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage 1Password credit card items.",
		Attributes:          attributes,
	}
}

// Create creates a credit card item.
func (r *OnePasswordCreditCard) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OnePasswordCreditCardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := r.client.GetVault(ctx, plan.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get vault", err.Error())
		return
	}

	inputs := []FieldInput{}
	addStringField(&inputs, "cardholder", onepassword.ItemFieldTypeText, plan.Cardholder)
	addStringField(&inputs, "number", onepassword.ItemFieldTypeCreditCardNumber, plan.Number)
	addStringField(&inputs, "type", onepassword.ItemFieldTypeCreditCardType, plan.Type)
	addStringField(&inputs, "expiry", onepassword.ItemFieldTypeMonthYear, plan.Expiry)
	extraFields := buildFieldsFromInputs(inputs, nil)

	params, err := buildItemCreateParamsFromShared(onepassword.ItemCategoryCreditCard, vaultID, plan.SharedItemModel, extraFields, nil, nil)
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

// Read refreshes state for a credit card item.
func (r *OnePasswordCreditCard) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OnePasswordCreditCardModel
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

// Update updates a credit card item.
func (r *OnePasswordCreditCard) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OnePasswordCreditCardModel
	var state OnePasswordCreditCardModel
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

	inputs := []FieldInput{}
	addStringField(&inputs, "cardholder", onepassword.ItemFieldTypeText, plan.Cardholder)
	addStringField(&inputs, "number", onepassword.ItemFieldTypeCreditCardNumber, plan.Number)
	addStringField(&inputs, "type", onepassword.ItemFieldTypeCreditCardType, plan.Type)
	addStringField(&inputs, "expiry", onepassword.ItemFieldTypeMonthYear, plan.Expiry)
	extraFields := buildFieldsFromInputs(inputs, existing)

	item, err := buildItemForUpdateFromShared(onepassword.ItemCategoryCreditCard, vaultID, plan.SharedItemModel, existing, extraFields, nil, nil)
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

// Delete removes a credit card item.
func (r *OnePasswordCreditCard) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OnePasswordCreditCardModel
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
