package onepasswordprovider

import (
	"context"
	"fmt"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnePasswordLoginModel represents the login resource.
type OnePasswordLoginModel struct {
	SharedItemModel
	Username types.String                          `tfsdk:"username"`
	Password OnePasswordSharedPasswordModel        `tfsdk:"password"`
	Websites map[string]OnePasswordWebsiteMapModel `tfsdk:"websites"`
}

// OnePasswordLogin manages login items.
type OnePasswordLogin struct {
	BaseItemResource
}

var (
	_ resource.ResourceWithConfigure = &OnePasswordLogin{}
)

// Metadata sets the resource type name.
func (r *OnePasswordLogin) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_login"
}

// Schema defines the schema for the login resource.
func (r *OnePasswordLogin) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := sharedItemAttributes()
	attributes["username"] = schema.StringAttribute{
		MarkdownDescription: "Login username.",
		Required:            true,
	}
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
		MarkdownDescription: "Manage 1Password login items.",
		Attributes:          attributes,
		Blocks: map[string]schema.Block{
			"password": schema.SingleNestedBlock{
				Blocks: map[string]schema.Block{
					"random": schema.SingleNestedBlock{
						Attributes: map[string]schema.Attribute{
							"length": schema.Int64Attribute{
								MarkdownDescription: "Password length.",
								Optional:            true,
							},
							"digits": schema.BoolAttribute{
								MarkdownDescription: "Include digits in password.",
								Optional:            true,
							},
							"symbols": schema.BoolAttribute{
								MarkdownDescription: "Include symbols in password.",
								Optional:            true,
							},
						},
					},
					"pin": schema.SingleNestedBlock{
						Attributes: map[string]schema.Attribute{
							"length": schema.Int64Attribute{
								MarkdownDescription: "Password length.",
								Optional:            true,
							},
						},
					},
					"memorable": schema.SingleNestedBlock{
						Attributes: map[string]schema.Attribute{
							"word_count": schema.Int64Attribute{
								MarkdownDescription: "Word count for password.",
								Optional:            true,
							},
							"word_list_type": schema.StringAttribute{
								MarkdownDescription: "Word list type for password.",
								Optional:            true,
							},
							"capitalize": schema.BoolAttribute{
								MarkdownDescription: "Capitalize one word in password.",
								Optional:            true,
							},
							"separator_type": schema.StringAttribute{
								MarkdownDescription: "Separator type for password.",
								Optional:            true,
							},
						},
					},
				},
			},
		},
	}
}

// Create creates a login item.
func (r *OnePasswordLogin) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OnePasswordLoginModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := r.client.GetVault(ctx, plan.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get vault", err.Error())
		return
	}

	password, err := r.generatePasswordFromPlan(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate password", err.Error())
		return
	}

	inputs := []FieldInput{}
	addStringField(&inputs, "username", onepassword.ItemFieldTypeText, plan.Username)
	addConcealedField(&inputs, "password", types.StringValue(password))
	extraFields := buildFieldsFromInputs(inputs, nil)

	websites, err := mapWebsitesFromMap(plan.Websites)
	if err != nil {
		resp.Diagnostics.AddError("Failed to map websites", err.Error())
		return
	}

	params, err := buildItemCreateParamsFromShared(onepassword.ItemCategoryLogin, vaultID, plan.SharedItemModel, extraFields, websites, nil)
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

func (r *OnePasswordLogin) generatePasswordFromPlan(ctx context.Context, plan OnePasswordLoginModel) (string, error) {
	recipe, err := buildPasswordRecipeFromShared(plan.Password)
	if err != nil {
		return "", err
	}

	password, err := r.client.GeneratePassword(ctx, recipe)
	if err != nil {
		return "", err
	}
	return password, nil
}

// Read refreshes state for a login item.
func (r *OnePasswordLogin) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OnePasswordLoginModel
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

// Update updates a login item.
func (r *OnePasswordLogin) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OnePasswordLoginModel
	var state OnePasswordLoginModel
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

	password, err := r.generatePasswordFromPlan(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate password", err.Error())
		return
	}

	inputs := []FieldInput{}
	addStringField(&inputs, "username", onepassword.ItemFieldTypeText, plan.Username)
	addStringField(&inputs, "password", onepassword.ItemFieldTypeConcealed, types.StringValue(password))
	extraFields := buildFieldsFromInputs(inputs, existing)

	websites, err := mapWebsitesFromMap(plan.Websites)
	if err != nil {
		resp.Diagnostics.AddError("Failed to map websites", err.Error())
		return
	}

	item, err := buildItemForUpdateFromShared(onepassword.ItemCategoryLogin, vaultID, plan.SharedItemModel, existing, extraFields, websites, nil)
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

// Delete removes a login item.
func (r *OnePasswordLogin) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OnePasswordLoginModel
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

func buildPasswordRecipeFromShared(data OnePasswordSharedPasswordModel) (onepassword.PasswordRecipe, error) {
	if data.Random != nil && data.Pin == nil && data.Memorable == nil {
		random := data.Random

		return onepassword.NewPasswordRecipeTypeVariantRandom(&onepassword.PasswordRecipeRandomInner{
			IncludeDigits:  random.Digits.ValueBool(),
			IncludeSymbols: random.Symbols.ValueBool(),
			Length:         uint32(random.Length.ValueInt64()),
		}), nil
	}

	if data.Pin != nil && data.Random == nil && data.Memorable == nil {
		pin := data.Pin

		return onepassword.NewPasswordRecipeTypeVariantPin(&onepassword.PasswordRecipePinInner{
			Length: uint32(pin.Length.ValueInt64()),
		}), nil
	}

	if data.Memorable != nil && data.Random == nil && data.Pin == nil {
		memorable := data.Memorable

		return onepassword.NewPasswordRecipeTypeVariantMemorable(&onepassword.PasswordRecipeMemorableInner{
			SeparatorType: onepassword.SeparatorType(memorable.SeparatorType.ValueString()),
			Capitalize:    memorable.Capitalize.ValueBool(),
			WordListType:  onepassword.WordListType(memorable.WordListType.ValueString()),
			WordCount:     uint32(memorable.WordCount.ValueInt64()),
		}), nil
	}

	return onepassword.PasswordRecipe{}, fmt.Errorf("one of random, pin or memorable must be set")
}
