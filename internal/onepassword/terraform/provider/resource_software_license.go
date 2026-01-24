package onepasswordprovider

import (
	"context"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnePasswordSoftwareLicenseSoftwareModel holds software fields.
type OnePasswordSoftwareLicenseSoftwareModel struct {
	Version    types.String `tfsdk:"version"`
	LicenseKey types.String `tfsdk:"license_key"`
}

// OnePasswordSoftwareLicenseCustomerModel holds customer fields.
type OnePasswordSoftwareLicenseCustomerModel struct {
	LicensedTo      types.String `tfsdk:"licensed_to"`
	RegisteredEmail types.String `tfsdk:"registered_email"`
	Company         types.String `tfsdk:"company"`
}

// OnePasswordSoftwareLicensePublisherModel holds publisher fields.
type OnePasswordSoftwareLicensePublisherModel struct {
	Name         types.String `tfsdk:"name"`
	DownloadPage types.String `tfsdk:"donwload_page"`
	Website      types.String `tfsdk:"website"`
	RetailPrice  types.Number `tfsdk:"retail_price"`
	SupportEmail types.String `tfsdk:"support_email"`
}

// OnePasswordSoftwareLicenseOrderModel holds order fields.
type OnePasswordSoftwareLicenseOrderModel struct {
	Number       types.String `tfsdk:"number"`
	Total        types.Number `tfsdk:"total"`
	PurchaseDate types.String `tfsdk:"purchase_date"`
}

// OnePasswordSoftwareLicenseModel represents software license items.
type OnePasswordSoftwareLicenseModel struct {
	SharedItemModel
	Software  *OnePasswordSoftwareLicenseSoftwareModel  `tfsdk:"software"`
	Customer  *OnePasswordSoftwareLicenseCustomerModel  `tfsdk:"customer"`
	Publisher *OnePasswordSoftwareLicensePublisherModel `tfsdk:"publisher"`
	Order     *OnePasswordSoftwareLicenseOrderModel     `tfsdk:"order"`
}

// OnePasswordSoftwareLicense manages software license items.
type OnePasswordSoftwareLicense struct {
	BaseItemResource
}

var (
	_ resource.ResourceWithConfigure = &OnePasswordSoftwareLicense{}
)

// Metadata sets the resource type name.
func (r *OnePasswordSoftwareLicense) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_software_license"
}

// Schema defines the schema for software license items.
func (r *OnePasswordSoftwareLicense) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := sharedItemAttributes()
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage 1Password software license items.",
		Attributes:          attributes,
		Blocks: map[string]schema.Block{
			"software": schema.SingleNestedBlock{
				MarkdownDescription: "Software details.",
				Attributes: map[string]schema.Attribute{
					"version": schema.StringAttribute{
						MarkdownDescription: "Software version.",
						Optional:            true,
					},
					"license_key": schema.StringAttribute{
						MarkdownDescription: "License key.",
						Optional:            true,
						Sensitive:           true,
						WriteOnly:           true,
					},
				},
			},
			"customer": schema.SingleNestedBlock{
				MarkdownDescription: "Customer details.",
				Attributes: map[string]schema.Attribute{
					"licensed_to": schema.StringAttribute{
						MarkdownDescription: "Licensed to.",
						Optional:            true,
					},
					"registered_email": schema.StringAttribute{
						MarkdownDescription: "Registered email.",
						Optional:            true,
					},
					"company": schema.StringAttribute{
						MarkdownDescription: "Company name.",
						Optional:            true,
					},
				},
			},
			"publisher": schema.SingleNestedBlock{
				MarkdownDescription: "Publisher details.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Publisher name.",
						Optional:            true,
					},
					"donwload_page": schema.StringAttribute{
						MarkdownDescription: "Download page URL.",
						Optional:            true,
					},
					"website": schema.StringAttribute{
						MarkdownDescription: "Publisher website.",
						Optional:            true,
					},
					"retail_price": schema.NumberAttribute{
						MarkdownDescription: "Retail price.",
						Optional:            true,
					},
					"support_email": schema.StringAttribute{
						MarkdownDescription: "Support email.",
						Optional:            true,
					},
				},
			},
			"order": schema.SingleNestedBlock{
				MarkdownDescription: "Order details.",
				Attributes: map[string]schema.Attribute{
					"number": schema.StringAttribute{
						MarkdownDescription: "Order number.",
						Optional:            true,
					},
					"total": schema.NumberAttribute{
						MarkdownDescription: "Order total.",
						Optional:            true,
					},
					"purchase_date": schema.StringAttribute{
						MarkdownDescription: "Purchase date.",
						Optional:            true,
					},
				},
			},
		},
	}
}

// Create creates a software license item.
func (r *OnePasswordSoftwareLicense) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OnePasswordSoftwareLicenseModel
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
	if plan.Software != nil {
		addStringField(&inputs, "software_version", onepassword.ItemFieldTypeText, plan.Software.Version)
		addStringField(&inputs, "software_license_key", onepassword.ItemFieldTypeText, plan.Software.LicenseKey)
	}
	if plan.Customer != nil {
		addStringField(&inputs, "customer_licensed_to", onepassword.ItemFieldTypeText, plan.Customer.LicensedTo)
		addStringField(&inputs, "customer_registered_email", onepassword.ItemFieldTypeEmail, plan.Customer.RegisteredEmail)
		addStringField(&inputs, "customer_company", onepassword.ItemFieldTypeText, plan.Customer.Company)
	}
	if plan.Publisher != nil {
		addStringField(&inputs, "publisher_name", onepassword.ItemFieldTypeText, plan.Publisher.Name)
		addStringField(&inputs, "publisher_download_page", onepassword.ItemFieldTypeURL, plan.Publisher.DownloadPage)
		addStringField(&inputs, "publisher_website", onepassword.ItemFieldTypeURL, plan.Publisher.Website)
		addNumberField(&inputs, "publisher_retail_price", onepassword.ItemFieldTypeText, plan.Publisher.RetailPrice)
		addStringField(&inputs, "publisher_support_email", onepassword.ItemFieldTypeEmail, plan.Publisher.SupportEmail)
	}
	if plan.Order != nil {
		addStringField(&inputs, "order_number", onepassword.ItemFieldTypeText, plan.Order.Number)
		addNumberField(&inputs, "order_total", onepassword.ItemFieldTypeText, plan.Order.Total)
		addStringField(&inputs, "order_purchase_date", onepassword.ItemFieldTypeText, plan.Order.PurchaseDate)
	}
	extraFields := buildFieldsFromInputs(inputs, nil)

	params, err := buildItemCreateParamsFromShared(onepassword.ItemCategorySoftwareLicense, vaultID, plan.SharedItemModel, r.defaultTags, extraFields, nil, nil)
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

// Read refreshes state for a software license item.
func (r *OnePasswordSoftwareLicense) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OnePasswordSoftwareLicenseModel
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

// Update updates a software license item.
func (r *OnePasswordSoftwareLicense) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OnePasswordSoftwareLicenseModel
	var state OnePasswordSoftwareLicenseModel
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
	if plan.Software != nil {
		addStringField(&inputs, "software_version", onepassword.ItemFieldTypeText, plan.Software.Version)
		addStringField(&inputs, "software_license_key", onepassword.ItemFieldTypeText, plan.Software.LicenseKey)
	}
	if plan.Customer != nil {
		addStringField(&inputs, "customer_licensed_to", onepassword.ItemFieldTypeText, plan.Customer.LicensedTo)
		addStringField(&inputs, "customer_registered_email", onepassword.ItemFieldTypeEmail, plan.Customer.RegisteredEmail)
		addStringField(&inputs, "customer_company", onepassword.ItemFieldTypeText, plan.Customer.Company)
	}
	if plan.Publisher != nil {
		addStringField(&inputs, "publisher_name", onepassword.ItemFieldTypeText, plan.Publisher.Name)
		addStringField(&inputs, "publisher_download_page", onepassword.ItemFieldTypeURL, plan.Publisher.DownloadPage)
		addStringField(&inputs, "publisher_website", onepassword.ItemFieldTypeURL, plan.Publisher.Website)
		addNumberField(&inputs, "publisher_retail_price", onepassword.ItemFieldTypeText, plan.Publisher.RetailPrice)
		addStringField(&inputs, "publisher_support_email", onepassword.ItemFieldTypeEmail, plan.Publisher.SupportEmail)
	}
	if plan.Order != nil {
		addStringField(&inputs, "order_number", onepassword.ItemFieldTypeText, plan.Order.Number)
		addNumberField(&inputs, "order_total", onepassword.ItemFieldTypeText, plan.Order.Total)
		addStringField(&inputs, "order_purchase_date", onepassword.ItemFieldTypeText, plan.Order.PurchaseDate)
	}
	extraFields := buildFieldsFromInputs(inputs, existing)

	item, err := buildItemForUpdateFromShared(onepassword.ItemCategorySoftwareLicense, vaultID, plan.SharedItemModel, existing, r.defaultTags, extraFields, nil, nil)
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

// Delete removes a software license item.
func (r *OnePasswordSoftwareLicense) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OnePasswordSoftwareLicenseModel
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
