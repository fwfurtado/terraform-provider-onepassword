package onepasswordprovider

import (
	"context"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnePasswordServerAdminConsoleModel holds admin console credentials.
type OnePasswordServerAdminConsoleModel struct {
	URL      types.String                   `tfsdk:"url"`
	Username types.String                   `tfsdk:"username"`
	Password OnePasswordSharedPasswordModel `tfsdk:"password"`
}

// OnePasswordServerSupportModel holds hosting provider support info.
type OnePasswordServerSupportModel struct {
	URL   types.String `tfsdk:"url"`
	Email types.String `tfsdk:"email"`
	Phone types.String `tfsdk:"phone"`
}

// OnePasswordServerHostingProviderModel holds hosting provider info.
type OnePasswordServerHostingProviderModel struct {
	Name    types.String                   `tfsdk:"name"`
	Website types.String                   `tfsdk:"website"`
	Support *OnePasswordServerSupportModel `tfsdk:"support"`
}

// OnePasswordServerModel represents server items.
type OnePasswordServerModel struct {
	SharedItemModel
	URL             types.String                           `tfsdk:"url"`
	Username        types.String                           `tfsdk:"username"`
	Password        OnePasswordSharedPasswordModel         `tfsdk:"password"`
	AdminConsole    *OnePasswordServerAdminConsoleModel    `tfsdk:"admin_console"`
	HostingProvider *OnePasswordServerHostingProviderModel `tfsdk:"hosting_provider"`
}

// OnePasswordServer manages server items.
type OnePasswordServer struct {
	BaseItemResource
}

var (
	_ resource.ResourceWithConfigure = &OnePasswordServer{}
)

// Metadata sets the resource type name.
func (r *OnePasswordServer) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

// Schema defines the schema for server items.
func (r *OnePasswordServer) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := sharedItemAttributes()
	attributes["url"] = schema.StringAttribute{
		MarkdownDescription: "Server URL.",
		Optional:            true,
	}
	attributes["username"] = schema.StringAttribute{
		MarkdownDescription: "Server username.",
		Optional:            true,
	}
	attributes["hosting_provider"] = schema.SingleNestedAttribute{
		MarkdownDescription: "Hosting provider details.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Hosting provider name.",
				Optional:            true,
			},
			"website": schema.StringAttribute{
				MarkdownDescription: "Hosting provider website.",
				Optional:            true,
			},
			"support": schema.SingleNestedAttribute{
				MarkdownDescription: "Support contact info.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						MarkdownDescription: "Support URL.",
						Optional:            true,
					},
					"email": schema.StringAttribute{
						MarkdownDescription: "Support email.",
						Optional:            true,
					},
					"phone": schema.StringAttribute{
						MarkdownDescription: "Support phone.",
						Optional:            true,
					},
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage 1Password server items.",
		Attributes:          attributes,
		Blocks: map[string]schema.Block{
			"password": passwordBlockSchema("Server password recipe."),
			"admin_console": schema.SingleNestedBlock{
				MarkdownDescription: "Admin console credentials.",
				Attributes: map[string]schema.Attribute{
					"url": schema.StringAttribute{
						MarkdownDescription: "Admin console URL.",
						Optional:            true,
					},
					"username": schema.StringAttribute{
						MarkdownDescription: "Admin console username.",
						Optional:            true,
					},
				},
				Blocks: map[string]schema.Block{
					"password": passwordBlockSchema("Admin console password recipe."),
				},
			},
		},
	}
}

// Create creates a server item.
func (r *OnePasswordServer) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OnePasswordServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := r.client.GetVault(ctx, plan.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get vault", err.Error())
		return
	}

	serverPassword, _, err := generatePasswordFromShared(ctx, r.client, &plan.Password, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate server password", err.Error())
		return
	}

	inputs := []FieldInput{}
	addStringField(&inputs, "url", onepassword.ItemFieldTypeURL, plan.URL)
	addStringField(&inputs, "username", onepassword.ItemFieldTypeText, plan.Username)
	addConcealedField(&inputs, "password", types.StringValue(serverPassword))
	if plan.AdminConsole != nil {
		adminPassword, _, err := generatePasswordFromShared(ctx, r.client, &plan.AdminConsole.Password, true)
		if err != nil {
			resp.Diagnostics.AddError("Failed to generate admin console password", err.Error())
			return
		}
		addStringField(&inputs, "admin_console_url", onepassword.ItemFieldTypeURL, plan.AdminConsole.URL)
		addStringField(&inputs, "admin_console_username", onepassword.ItemFieldTypeText, plan.AdminConsole.Username)
		addConcealedField(&inputs, "admin_console_password", types.StringValue(adminPassword))
	}
	if plan.HostingProvider != nil {
		addStringField(&inputs, "hosting_provider_name", onepassword.ItemFieldTypeText, plan.HostingProvider.Name)
		addStringField(&inputs, "hosting_provider_website", onepassword.ItemFieldTypeURL, plan.HostingProvider.Website)
		if plan.HostingProvider.Support != nil {
			addStringField(&inputs, "support_url", onepassword.ItemFieldTypeURL, plan.HostingProvider.Support.URL)
			addStringField(&inputs, "support_email", onepassword.ItemFieldTypeEmail, plan.HostingProvider.Support.Email)
			addStringField(&inputs, "support_phone", onepassword.ItemFieldTypePhone, plan.HostingProvider.Support.Phone)
		}
	}
	extraFields := buildFieldsFromInputs(inputs, nil)

	params, err := buildItemCreateParamsFromShared(onepassword.ItemCategoryServer, vaultID, plan.SharedItemModel, extraFields, nil, nil)
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

// Read refreshes state for a server item.
func (r *OnePasswordServer) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OnePasswordServerModel
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

// Update updates a server item.
func (r *OnePasswordServer) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OnePasswordServerModel
	var state OnePasswordServerModel
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

	serverPassword, _, err := generatePasswordFromShared(ctx, r.client, &plan.Password, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate server password", err.Error())
		return
	}

	inputs := []FieldInput{}
	addStringField(&inputs, "url", onepassword.ItemFieldTypeURL, plan.URL)
	addStringField(&inputs, "username", onepassword.ItemFieldTypeText, plan.Username)
	addConcealedField(&inputs, "password", types.StringValue(serverPassword))
	if plan.AdminConsole != nil {
		adminPassword, _, err := generatePasswordFromShared(ctx, r.client, &plan.AdminConsole.Password, true)
		if err != nil {
			resp.Diagnostics.AddError("Failed to generate admin console password", err.Error())
			return
		}
		addStringField(&inputs, "admin_console_url", onepassword.ItemFieldTypeURL, plan.AdminConsole.URL)
		addStringField(&inputs, "admin_console_username", onepassword.ItemFieldTypeText, plan.AdminConsole.Username)
		addConcealedField(&inputs, "admin_console_password", types.StringValue(adminPassword))
	}
	if plan.HostingProvider != nil {
		addStringField(&inputs, "hosting_provider_name", onepassword.ItemFieldTypeText, plan.HostingProvider.Name)
		addStringField(&inputs, "hosting_provider_website", onepassword.ItemFieldTypeURL, plan.HostingProvider.Website)
		if plan.HostingProvider.Support != nil {
			addStringField(&inputs, "support_url", onepassword.ItemFieldTypeURL, plan.HostingProvider.Support.URL)
			addStringField(&inputs, "support_email", onepassword.ItemFieldTypeEmail, plan.HostingProvider.Support.Email)
			addStringField(&inputs, "support_phone", onepassword.ItemFieldTypePhone, plan.HostingProvider.Support.Phone)
		}
	}
	extraFields := buildFieldsFromInputs(inputs, existing)

	item, err := buildItemForUpdateFromShared(onepassword.ItemCategoryServer, vaultID, plan.SharedItemModel, existing, extraFields, nil, nil)
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

// Delete removes a server item.
func (r *OnePasswordServer) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OnePasswordServerModel
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
