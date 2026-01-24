package onepasswordprovider

import (
	"context"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnePasswordRouterBaseStationModel holds base station credentials.
type OnePasswordRouterBaseStationModel struct {
	Name     types.String                   `tfsdk:"name"`
	Password OnePasswordSharedPasswordModel `tfsdk:"password"`
}

// OnePasswordRouterWirelessModel holds wireless settings.
type OnePasswordRouterWirelessModel struct {
	SecurityType types.String                   `tfsdk:"security_type"`
	Passphrase   OnePasswordSharedPasswordModel `tfsdk:"passphrase"`
}

// OnePasswordRouterModel represents router items.
type OnePasswordRouterModel struct {
	SharedItemModel
	ServerIP                types.String                       `tfsdk:"server_ip"`
	AirportID               types.String                       `tfsdk:"airport_id"`
	NetworkName             types.String                       `tfsdk:"network_name"`
	AttachedStoragePassword OnePasswordSharedPasswordModel     `tfsdk:"attached_storage_password"`
	BaseStation             *OnePasswordRouterBaseStationModel `tfsdk:"base_station"`
	Wireless                *OnePasswordRouterWirelessModel    `tfsdk:"wireless"`
}

// OnePasswordRouter manages router items.
type OnePasswordRouter struct {
	BaseItemResource
}

var (
	_ resource.ResourceWithConfigure = &OnePasswordRouter{}
)

// Metadata sets the resource type name.
func (r *OnePasswordRouter) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_router"
}

// Schema defines the schema for router items.
func (r *OnePasswordRouter) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := sharedItemAttributes()
	attributes["server_ip"] = schema.StringAttribute{
		MarkdownDescription: "Router server IP.",
		Optional:            true,
	}
	attributes["airport_id"] = schema.StringAttribute{
		MarkdownDescription: "Airport ID.",
		Optional:            true,
	}
	attributes["network_name"] = schema.StringAttribute{
		MarkdownDescription: "Network name.",
		Optional:            true,
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage 1Password router items.",
		Attributes:          attributes,
		Blocks: map[string]schema.Block{
			"attached_storage_password": passwordBlockSchema("Attached storage password recipe."),
			"base_station": schema.SingleNestedBlock{
				MarkdownDescription: "Base station credentials.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Base station name.",
						Optional:            true,
					},
				},
				Blocks: map[string]schema.Block{
					"password": passwordBlockSchema("Base station password recipe."),
				},
			},
			"wireless": schema.SingleNestedBlock{
				MarkdownDescription: "Wireless settings.",
				Attributes: map[string]schema.Attribute{
					"security_type": schema.StringAttribute{
						MarkdownDescription: "Wireless security type.",
						Optional:            true,
					},
				},
				Blocks: map[string]schema.Block{
					"passphrase": passwordBlockSchema("Wireless passphrase recipe."),
				},
			},
		},
	}
}

// Create creates a router item.
func (r *OnePasswordRouter) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OnePasswordRouterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := r.client.GetVault(ctx, plan.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get vault", err.Error())
		return
	}

	attachedStoragePassword, _, err := generatePasswordFromShared(ctx, r.client, &plan.AttachedStoragePassword, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate attached storage password", err.Error())
		return
	}

	inputs := []FieldInput{}
	addStringField(&inputs, "server_ip", onepassword.ItemFieldTypeText, plan.ServerIP)
	addStringField(&inputs, "airport_id", onepassword.ItemFieldTypeText, plan.AirportID)
	addStringField(&inputs, "network_name", onepassword.ItemFieldTypeText, plan.NetworkName)
	addConcealedField(&inputs, "attached_storage_password", types.StringValue(attachedStoragePassword))
	if plan.BaseStation != nil {
		baseStationPassword, _, err := generatePasswordFromShared(ctx, r.client, &plan.BaseStation.Password, true)
		if err != nil {
			resp.Diagnostics.AddError("Failed to generate base station password", err.Error())
			return
		}
		addStringField(&inputs, "base_station_name", onepassword.ItemFieldTypeText, plan.BaseStation.Name)
		addConcealedField(&inputs, "base_station_password", types.StringValue(baseStationPassword))
	}
	if plan.Wireless != nil {
		wirelessPassphrase, _, err := generatePasswordFromShared(ctx, r.client, &plan.Wireless.Passphrase, true)
		if err != nil {
			resp.Diagnostics.AddError("Failed to generate wireless passphrase", err.Error())
			return
		}
		addStringField(&inputs, "wireless_security_type", onepassword.ItemFieldTypeText, plan.Wireless.SecurityType)
		addConcealedField(&inputs, "wireless_passphrase", types.StringValue(wirelessPassphrase))
	}
	extraFields := buildFieldsFromInputs(inputs, nil)

	params, err := buildItemCreateParamsFromShared(onepassword.ItemCategoryRouter, vaultID, plan.SharedItemModel, r.defaultTags, extraFields, nil, nil)
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

// Read refreshes state for a router item.
func (r *OnePasswordRouter) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OnePasswordRouterModel
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

// Update updates a router item.
func (r *OnePasswordRouter) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OnePasswordRouterModel
	var state OnePasswordRouterModel
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

	attachedStoragePassword, _, err := generatePasswordFromShared(ctx, r.client, &plan.AttachedStoragePassword, true)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate attached storage password", err.Error())
		return
	}

	inputs := []FieldInput{}
	addStringField(&inputs, "server_ip", onepassword.ItemFieldTypeText, plan.ServerIP)
	addStringField(&inputs, "airport_id", onepassword.ItemFieldTypeText, plan.AirportID)
	addStringField(&inputs, "network_name", onepassword.ItemFieldTypeText, plan.NetworkName)
	addConcealedField(&inputs, "attached_storage_password", types.StringValue(attachedStoragePassword))
	if plan.BaseStation != nil {
		baseStationPassword, _, err := generatePasswordFromShared(ctx, r.client, &plan.BaseStation.Password, true)
		if err != nil {
			resp.Diagnostics.AddError("Failed to generate base station password", err.Error())
			return
		}
		addStringField(&inputs, "base_station_name", onepassword.ItemFieldTypeText, plan.BaseStation.Name)
		addConcealedField(&inputs, "base_station_password", types.StringValue(baseStationPassword))
	}
	if plan.Wireless != nil {
		wirelessPassphrase, _, err := generatePasswordFromShared(ctx, r.client, &plan.Wireless.Passphrase, true)
		if err != nil {
			resp.Diagnostics.AddError("Failed to generate wireless passphrase", err.Error())
			return
		}
		addStringField(&inputs, "wireless_security_type", onepassword.ItemFieldTypeText, plan.Wireless.SecurityType)
		addConcealedField(&inputs, "wireless_passphrase", types.StringValue(wirelessPassphrase))
	}
	extraFields := buildFieldsFromInputs(inputs, existing)

	item, err := buildItemForUpdateFromShared(onepassword.ItemCategoryRouter, vaultID, plan.SharedItemModel, existing, r.defaultTags, extraFields, nil, nil)
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

// Delete removes a router item.
func (r *OnePasswordRouter) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OnePasswordRouterModel
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
