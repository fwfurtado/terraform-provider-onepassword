package onepasswordprovider

import (
	"context"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnePasswordDocumentResourceModel represents document items.
type OnePasswordDocumentResourceModel struct {
	SharedItemModel
	Document *OnePasswordDocumentModel `tfsdk:"document"`
}

// OnePasswordDocument manages document items.
type OnePasswordDocument struct {
	BaseItemResource
}

var (
	_ resource.ResourceWithConfigure = &OnePasswordDocument{}
)

// Metadata sets the resource type name.
func (r *OnePasswordDocument) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_document"
}

// Schema defines the schema for document items.
func (r *OnePasswordDocument) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage 1Password document items.",
		Attributes:          sharedItemAttributes(),
		Blocks: map[string]schema.Block{
			"document": schema.SingleNestedBlock{
				MarkdownDescription: "Document content.",
				Attributes: map[string]schema.Attribute{
					"filename": schema.StringAttribute{
						MarkdownDescription: "Document filename.",
						Required:            true,
					},
					"content": schema.StringAttribute{
						MarkdownDescription: "Document content.",
						Required:            true,
						Sensitive:           true,
					},
				},
			},
		},
	}
}

// Create creates a document item.
func (r *OnePasswordDocument) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OnePasswordDocumentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := r.client.GetVault(ctx, plan.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get vault", err.Error())
		return
	}

	if plan.Document == nil {
		resp.Diagnostics.AddError("Missing document", "document block is required")
		return
	}

	document, err := mapDocument(plan.Document)
	if err != nil {
		resp.Diagnostics.AddError("Failed to map document", err.Error())
		return
	}

	params, err := buildItemCreateParamsFromShared(onepassword.ItemCategoryDocument, vaultID, plan.SharedItemModel, nil, nil, document)
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

// Read refreshes state for a document item.
func (r *OnePasswordDocument) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OnePasswordDocumentResourceModel
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

// Update updates a document item.
func (r *OnePasswordDocument) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OnePasswordDocumentResourceModel
	var state OnePasswordDocumentResourceModel
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

	if plan.Document == nil {
		resp.Diagnostics.AddError("Missing document", "document block is required")
		return
	}

	document, err := mapDocument(plan.Document)
	if err != nil {
		resp.Diagnostics.AddError("Failed to map document", err.Error())
		return
	}

	existing, err := r.client.GetItem(ctx, vaultID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to load existing item", err.Error())
		return
	}

	item, err := buildItemForUpdateFromShared(onepassword.ItemCategoryDocument, vaultID, plan.SharedItemModel, existing, nil, nil, document)
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

// Delete removes a document item.
func (r *OnePasswordDocument) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OnePasswordDocumentResourceModel
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
