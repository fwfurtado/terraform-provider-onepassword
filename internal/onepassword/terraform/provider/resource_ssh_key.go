package onepasswordprovider

import (
	"context"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnePasswordSSHKeyModel represents SSH key items.
type OnePasswordSSHKeyModel struct {
	SharedItemModel
	PrivateKey  types.String `tfsdk:"private_key"`
	PublicKey   types.String `tfsdk:"public_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	KeyType     types.String `tfsdk:"key_type"`
}

// OnePasswordSSHKey manages SSH key items.
type OnePasswordSSHKey struct {
	BaseItemResource
}

var (
	_ resource.ResourceWithConfigure = &OnePasswordSSHKey{}
)

// Metadata sets the resource type name.
func (r *OnePasswordSSHKey) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

// Schema defines the schema for SSH key items.
func (r *OnePasswordSSHKey) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := sharedItemAttributes()
	attributes["private_key"] = schema.StringAttribute{
		MarkdownDescription: "SSH private key.",
		Required:            true,
		Sensitive:           true,
		WriteOnly:           true,
	}
	attributes["public_key"] = schema.StringAttribute{
		MarkdownDescription: "SSH public key.",
		Computed:            true,
	}
	attributes["fingerprint"] = schema.StringAttribute{
		MarkdownDescription: "SSH key fingerprint.",
		Computed:            true,
	}
	attributes["key_type"] = schema.StringAttribute{
		MarkdownDescription: "SSH key type.",
		Computed:            true,
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage 1Password SSH key items.",
		Attributes:          attributes,
	}
}

// Create creates an SSH key item.
func (r *OnePasswordSSHKey) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OnePasswordSSHKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := r.client.GetVault(ctx, plan.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get vault", err.Error())
		return
	}

	privateKey, ok := getOptionalString(plan.PrivateKey)
	if !ok {
		resp.Diagnostics.AddError("Missing private key", "private_key is required")
		return
	}

	publicKey, fingerprint, keyType, err := parseSSHKeyDetails(privateKey)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse SSH key", err.Error())
		return
	}

	inputs := []FieldInput{{
		Label: "private key",
		Type:  onepassword.ItemFieldTypeSSHKey,
		Value: privateKey,
		ID:    "private_key",
		Details: func() *onepassword.ItemFieldDetails {
			attrs := onepassword.SSHKeyAttributes{
				PublicKey:   publicKey,
				Fingerprint: fingerprint,
				KeyType:     keyType,
			}
			details := onepassword.NewItemFieldDetailsTypeVariantSSHKey(&attrs)
			return &details
		}(),
	}}
	extraFields := buildFieldsFromInputs(inputs, nil)

	params, err := buildItemCreateParamsFromShared(onepassword.ItemCategorySSHKey, vaultID, plan.SharedItemModel, extraFields, nil, nil)
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
	plan.PublicKey = types.StringValue(publicKey)
	plan.Fingerprint = types.StringValue(fingerprint)
	plan.KeyType = types.StringValue(keyType)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes state for an SSH key item.
func (r *OnePasswordSSHKey) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OnePasswordSSHKeyModel
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

// Update updates an SSH key item.
func (r *OnePasswordSSHKey) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OnePasswordSSHKeyModel
	var state OnePasswordSSHKeyModel
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

	privateKey, ok := getOptionalString(plan.PrivateKey)
	if !ok {
		resp.Diagnostics.AddError("Missing private key", "private_key is required")
		return
	}

	publicKey, fingerprint, keyType, err := parseSSHKeyDetails(privateKey)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse SSH key", err.Error())
		return
	}

	existing, err := r.client.GetItem(ctx, vaultID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to load existing item", err.Error())
		return
	}

	inputs := []FieldInput{{
		Label: "private key",
		Type:  onepassword.ItemFieldTypeSSHKey,
		Value: privateKey,
		ID:    "private_key",
		Details: func() *onepassword.ItemFieldDetails {
			attrs := onepassword.SSHKeyAttributes{
				PublicKey:   publicKey,
				Fingerprint: fingerprint,
				KeyType:     keyType,
			}
			details := onepassword.NewItemFieldDetailsTypeVariantSSHKey(&attrs)
			return &details
		}(),
	}}
	extraFields := buildFieldsFromInputs(inputs, existing)

	item, err := buildItemForUpdateFromShared(onepassword.ItemCategorySSHKey, vaultID, plan.SharedItemModel, existing, extraFields, nil, nil)
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
	plan.PublicKey = types.StringValue(publicKey)
	plan.Fingerprint = types.StringValue(fingerprint)
	plan.KeyType = types.StringValue(keyType)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes an SSH key item.
func (r *OnePasswordSSHKey) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OnePasswordSSHKeyModel
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
