package onepasswordprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/1password/onepassword-sdk-go"
	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnePasswordResourceItemModel is the Terraform model for a 1Password item.
type OnePasswordResourceItemModel struct {
	ID              types.String                          `tfsdk:"id"`
	Vault           types.String                          `tfsdk:"vault"`
	Name            types.String                          `tfsdk:"name"`
	Category        types.String                          `tfsdk:"category"`
	Username        types.String                          `tfsdk:"username"`
	Password        types.String                          `tfsdk:"password"`
	PasswordVersion types.Int64                           `tfsdk:"password_version"`
	Websites        []OnePasswordResourceItemWebsiteModel `tfsdk:"websites"`
	Tags            []types.String                        `tfsdk:"tags"`
	Version         types.Int64                           `tfsdk:"version"`
	Note            types.String                          `tfsdk:"note"`
	Notes           types.String                          `tfsdk:"notes"`
	Document        *OnePasswordResourceItemDocumentModel `tfsdk:"document"`
	Sections        *OnePasswordSectionMapModel           `tfsdk:"sections"`
}

// OnePasswordResourceItemWebsiteModel maps item website attributes.
type OnePasswordResourceItemWebsiteModel struct {
	URL              types.String `tfsdk:"url"`
	Label            types.String `tfsdk:"label"`
	AutofillBehavior types.String `tfsdk:"autofill_behavior"`
}

// OnePasswordResourceItemDocumentModel holds a document attachment.
type OnePasswordResourceItemDocumentModel struct {
	Name    types.String `tfsdk:"name"`
	Content types.String `tfsdk:"content"`
}

// OnePasswordResourceItemSectionModel represents an item section.
type OnePasswordResourceItemSectionModel struct {
	Label  types.String                        `tfsdk:"label"`
	Fields []OnePasswordResourceItemFieldModel `tfsdk:"fields"`
}

// OnePasswordResourceItemFieldModel represents a field in an item section.
type OnePasswordResourceItemFieldModel struct {
	Label    types.String                         `tfsdk:"label"`
	ID       types.String                         `tfsdk:"id"`
	Type     types.String                         `tfsdk:"type"`
	Value    types.String                         `tfsdk:"value"`
	Metadata types.Map                            `tfsdk:"metadata"`
	File     *OnePasswordResourceItemFileModel    `tfsdk:"file"`
	Address  *OnePasswordResourceItemAddressModel `tfsdk:"address"`
	SSHKey   *OnePasswordResourceItemSSHKeyModel  `tfsdk:"ssh_key"`
	TOTP     *OnePasswordResourceItemTOTPModel    `tfsdk:"totp"`
}

// OnePasswordResourceItemFileModel represents a file attachment for a field.
type OnePasswordResourceItemFileModel struct {
	Name    types.String `tfsdk:"name"`
	Content types.String `tfsdk:"content"`
}

// OnePasswordResourceItemAddressModel holds address field details.
type OnePasswordResourceItemAddressModel struct {
	Street  types.String `tfsdk:"street"`
	City    types.String `tfsdk:"city"`
	State   types.String `tfsdk:"state"`
	Zip     types.String `tfsdk:"zip"`
	Country types.String `tfsdk:"country"`
}

// OnePasswordResourceItemSSHKeyModel holds SSH key field details.
type OnePasswordResourceItemSSHKeyModel struct {
	PublicKey   types.String `tfsdk:"public_key"`
	PrivateKey  types.String `tfsdk:"private_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	KeyType     types.String `tfsdk:"key_type"`
}

// OnePasswordResourceItemTOTPModel holds TOTP field details.
type OnePasswordResourceItemTOTPModel struct {
	Code         types.String `tfsdk:"code"`
	ErrorMessage types.String `tfsdk:"error_message"`
}

// OnePasswordResourceItem manages 1Password items as a Terraform resource.
type OnePasswordResourceItem struct {
	client *client.ClientWrapper
}

var (
	_ resource.ResourceWithConfigure = &OnePasswordResourceItem{}
)

// Metadata sets the resource type name.
func (r *OnePasswordResourceItem) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_item"
}

// Configure stores the configured 1Password client.
func (r *OnePasswordResourceItem) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clientWrapper, ok := req.ProviderData.(*client.ClientWrapper)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *ClientWrapper, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if clientWrapper == nil {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			"The 1Password client is required but was not configured. Please report this issue to the provider developers.",
		)
		return
	}

	r.client = clientWrapper
}

// Schema defines the schema for the onepassword_item resource.
func (r *OnePasswordResourceItem) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage 1Password items.\n\n" +
			"Example:\n" +
			"```hcl\n" +
			"resource \"onepassword_item\" \"app_login\" {\n" +
			"  vault    = \"Engineering\"\n" +
			"  name     = \"App Login\"\n" +
			"  category = \"login\"\n" +
			"\n" +
			"  username = \"app-user\"\n" +
			"  password = var.app_password\n" +
			"\n" +
			"  websites = [\n" +
			"    {\n" +
			"      url               = \"https://app.example.com\"\n" +
			"      label             = \"App\"\n" +
			"      autofill_behavior = \"exact-domain\"\n" +
			"    }\n" +
			"  ]\n" +
			"\n" +
			"  notes = \"Managed by Terraform\"\n" +
			"  tags  = [\"terraform\", \"app\"]\n" +
			"}\n" +
			"```\n\n" +
			"References:\n" +
			"- https://developer.1password.com/docs/cli/item-categories/\n" +
			"- https://developer.hashicorp.com/terraform/language/resources/syntax",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Item ID.",
				Computed:            true,
			},
			"vault": schema.StringAttribute{
				MarkdownDescription: "1Password vault title.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Item title.",
				Required:            true,
			},
			"category": schema.StringAttribute{
				MarkdownDescription: "Item category. Use 1Password item category names (e.g. login, secure-note, password).\n\n" +
					"Reference: https://developer.1password.com/docs/cli/item-categories/",
				Required: true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username field value.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password field value.",
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
			},
			"password_version": schema.Int64Attribute{
				MarkdownDescription: "Version used to force password rotation.",
				Optional:            true,
			},
			"notes": schema.StringAttribute{
				MarkdownDescription: "Item notes.",
				Optional:            true,
			},
			"note": schema.StringAttribute{
				MarkdownDescription: "Item note.",
				Optional:            true,
			},
			"tags": schema.ListAttribute{
				MarkdownDescription: "Item tags.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"version": schema.Int64Attribute{
				MarkdownDescription: "Version trigger to refresh ephemeral values.",
				Optional:            true,
			},
			"websites": schema.ListNestedAttribute{
				MarkdownDescription: "Websites associated with the item.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"url": schema.StringAttribute{
							MarkdownDescription: "Website URL.",
							Required:            true,
						},
						"label": schema.StringAttribute{
							MarkdownDescription: "Website label.",
							Optional:            true,
						},
						"autofill_behavior": schema.StringAttribute{
							MarkdownDescription: "Website autofill behavior.",
							Optional:            true,
						},
					},
				},
			},
			"document": schema.SingleNestedAttribute{
				MarkdownDescription: "Document attachment for the item.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Document file name.",
						Required:            true,
					},
					"content": schema.StringAttribute{
						MarkdownDescription: "Document file content.",
						Required:            true,
						Sensitive:           true,
						WriteOnly:           true,
					},
				},
			},
			"sections": sectionsAttribute("Item sections map."),
		},
	}
}

// Create creates a 1Password item.
func (r *OnePasswordResourceItem) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OnePasswordResourceItemModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := r.client.GetVault(ctx, plan.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get vault", err.Error())
		return
	}

	params, err := buildItemCreateParams(&plan, vaultID, nil)
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

// Read refreshes the Terraform state for a 1Password item.
func (r *OnePasswordResourceItem) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OnePasswordResourceItemModel
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

	_, err = r.client.GetItem(ctx, vaultID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read item", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates a 1Password item.
func (r *OnePasswordResourceItem) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OnePasswordResourceItemModel
	var state OnePasswordResourceItemModel
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

	existingItem, err := r.client.GetItem(ctx, vaultID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to load existing item", err.Error())
		return
	}

	item, err := buildItemForUpdate(&plan, vaultID, existingItem)
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

// Delete removes a 1Password item.
func (r *OnePasswordResourceItem) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OnePasswordResourceItemModel
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

// buildItemCreateParams maps the Terraform model into the SDK create parameters.
func buildItemCreateParams(plan *OnePasswordResourceItemModel, vaultID string, existing *onepassword.Item) (onepassword.ItemCreateParams, error) {
	category, err := mapItemCategory(plan.Category)
	if err != nil {
		return onepassword.ItemCreateParams{}, err
	}

	sections, fields, files, err := buildItemSectionsAndFields(plan, existing)
	if err != nil {
		return onepassword.ItemCreateParams{}, err
	}

	params := onepassword.ItemCreateParams{
		Category: category,
		VaultID:  vaultID,
		Title:    plan.Name.ValueString(),
		Fields:   fields,
		Sections: sections,
		Files:    files,
	}

	if note := noteFromShared(SharedItemModel{Note: plan.Note, Notes: plan.Notes}); note != nil {
		params.Notes = note
	}

	if len(plan.Tags) > 0 {
		params.Tags = mapTags(plan.Tags)
	}

	if len(plan.Websites) > 0 {
		websites, err := mapWebsites(plan.Websites)
		if err != nil {
			return onepassword.ItemCreateParams{}, err
		}
		params.Websites = websites
	}

	if plan.Document != nil {
		document, err := mapDocument(plan.Document)
		if err != nil {
			return onepassword.ItemCreateParams{}, err
		}
		params.Document = document
	}

	return params, nil
}

// buildItemForUpdate maps the Terraform model into an SDK update item.
func buildItemForUpdate(plan *OnePasswordResourceItemModel, vaultID string, existing *onepassword.Item) (onepassword.Item, error) {
	category, err := mapItemCategory(plan.Category)
	if err != nil {
		return onepassword.Item{}, err
	}

	sections, fields, files, err := buildItemSectionsAndFields(plan, existing)
	if err != nil {
		return onepassword.Item{}, err
	}

	item := onepassword.Item{
		ID:       existing.ID,
		VaultID:  vaultID,
		Title:    plan.Name.ValueString(),
		Category: category,
		Fields:   fields,
		Sections: sections,
		Files:    convertFilesToItemFiles(files),
		Version:  existing.Version,
	}

	if note := noteFromShared(SharedItemModel{Note: plan.Note, Notes: plan.Notes}); note != nil {
		item.Notes = *note
	}

	if len(plan.Tags) > 0 {
		item.Tags = mapTags(plan.Tags)
	}

	if len(plan.Websites) > 0 {
		websites, err := mapWebsites(plan.Websites)
		if err != nil {
			return onepassword.Item{}, err
		}
		item.Websites = websites
	}

	if plan.Document != nil {
		document, err := mapDocument(plan.Document)
		if err != nil {
			return onepassword.Item{}, err
		}
		item.Document = &onepassword.FileAttributes{Name: document.Name}
	}

	return item, nil
}

// buildItemSectionsAndFields turns sections/fields into 1Password SDK structures.
func buildItemSectionsAndFields(plan *OnePasswordResourceItemModel, existing *onepassword.Item) ([]onepassword.ItemSection, []onepassword.ItemField, []onepassword.FileCreateParams, error) {
	sections, fields, files, err := buildSectionsFromMap(plan.Sections, existing)
	if err != nil {
		return nil, nil, nil, err
	}

	if username, ok := getOptionalString(plan.Username); ok {
		fields = append(fields, onepassword.ItemField{
			ID:        "username",
			Title:     "username",
			FieldType: onepassword.ItemFieldTypeText,
			Value:     username,
		})
	}

	if password, ok := getOptionalString(plan.Password); ok {
		fields = append(fields, onepassword.ItemField{
			ID:        "password",
			Title:     "password",
			FieldType: onepassword.ItemFieldTypeConcealed,
			Value:     password,
		})
	}

	return sections, fields, files, nil
}

// mapItemCategory normalizes the category string to the SDK enum.
func mapItemCategory(value types.String) (onepassword.ItemCategory, error) {
	if value.IsNull() || value.IsUnknown() {
		return "", fmt.Errorf("category is required")
	}

	switch strings.ToLower(value.ValueString()) {
	case "login":
		return onepassword.ItemCategoryLogin, nil
	case "secure-note":
		return onepassword.ItemCategorySecureNote, nil
	case "credit-card":
		return onepassword.ItemCategoryCreditCard, nil
	case "crypto-wallet":
		return onepassword.ItemCategoryCryptoWallet, nil
	case "identity":
		return onepassword.ItemCategoryIdentity, nil
	case "password":
		return onepassword.ItemCategoryPassword, nil
	case "document":
		return onepassword.ItemCategoryDocument, nil
	case "api-credentials":
		return onepassword.ItemCategoryAPICredentials, nil
	case "bank-account":
		return onepassword.ItemCategoryBankAccount, nil
	case "database":
		return onepassword.ItemCategoryDatabase, nil
	case "driver-license":
		return onepassword.ItemCategoryDriverLicense, nil
	case "email":
		return onepassword.ItemCategoryEmail, nil
	case "medical-record":
		return onepassword.ItemCategoryMedicalRecord, nil
	case "membership":
		return onepassword.ItemCategoryMembership, nil
	case "outdoor-license":
		return onepassword.ItemCategoryOutdoorLicense, nil
	case "passport":
		return onepassword.ItemCategoryPassport, nil
	case "rewards":
		return onepassword.ItemCategoryRewards, nil
	case "router":
		return onepassword.ItemCategoryRouter, nil
	case "server":
		return onepassword.ItemCategoryServer, nil
	case "ssh-key":
		return onepassword.ItemCategorySSHKey, nil
	case "social-security-number":
		return onepassword.ItemCategorySocialSecurityNumber, nil
	case "software-license":
		return onepassword.ItemCategorySoftwareLicense, nil
	case "person":
		return onepassword.ItemCategoryPerson, nil
	case "unsupported":
		return onepassword.ItemCategoryUnsupported, nil
	default:
		return "", fmt.Errorf("unsupported category: %s", value.ValueString())
	}
}

// mapFieldType normalizes the field type string to the SDK enum.
func mapFieldType(value types.String) (onepassword.ItemFieldType, error) {
	if value.IsNull() || value.IsUnknown() {
		return "", fmt.Errorf("field type is required")
	}

	switch strings.ToLower(value.ValueString()) {
	case "text":
		return onepassword.ItemFieldTypeText, nil
	case "concealed":
		return onepassword.ItemFieldTypeConcealed, nil
	case "credit-card-number":
		return onepassword.ItemFieldTypeCreditCardNumber, nil
	case "credit-card-type":
		return onepassword.ItemFieldTypeCreditCardType, nil
	case "phone":
		return onepassword.ItemFieldTypePhone, nil
	case "url":
		return onepassword.ItemFieldTypeURL, nil
	case "totp":
		return onepassword.ItemFieldTypeTOTP, nil
	case "email":
		return onepassword.ItemFieldTypeEmail, nil
	case "reference":
		return onepassword.ItemFieldTypeReference, nil
	case "ssh-key":
		return onepassword.ItemFieldTypeSSHKey, nil
	case "menu":
		return onepassword.ItemFieldTypeMenu, nil
	case "month-year":
		return onepassword.ItemFieldTypeMonthYear, nil
	case "address":
		return onepassword.ItemFieldTypeAddress, nil
	case "date":
		return onepassword.ItemFieldTypeDate, nil
	case "unsupported":
		return onepassword.ItemFieldTypeUnsupported, nil
	default:
		return "", fmt.Errorf("unsupported field type: %s", value.ValueString())
	}
}

// mapWebsites converts Terraform website models to SDK website entries.
func mapWebsites(websites []OnePasswordResourceItemWebsiteModel) ([]onepassword.Website, error) {
	result := make([]onepassword.Website, 0, len(websites))
	for _, website := range websites {
		if website.URL.IsNull() || website.URL.IsUnknown() {
			return nil, fmt.Errorf("website url is required")
		}

		autofill := onepassword.AutofillBehaviorAnywhereOnWebsite
		if value, ok := getOptionalString(website.AutofillBehavior); ok {
			mapped, err := mapAutofillBehavior(value)
			if err != nil {
				return nil, err
			}
			autofill = mapped
		}

		result = append(result, onepassword.Website{
			URL:              website.URL.ValueString(),
			Label:            valueOrEmpty(website.Label),
			AutofillBehavior: autofill,
		})
	}

	return result, nil
}

// mapAutofillBehavior converts the HCL string into the SDK autofill behavior.
func mapAutofillBehavior(value string) (onepassword.AutofillBehavior, error) {
	switch strings.ToLower(value) {
	case "anywhere-on-website":
		return onepassword.AutofillBehaviorAnywhereOnWebsite, nil
	case "exact-domain":
		return onepassword.AutofillBehaviorExactDomain, nil
	case "never":
		return onepassword.AutofillBehaviorNever, nil
	default:
		return "", fmt.Errorf("unsupported autofill behavior: %s", value)
	}
}

// mapDocument converts a Terraform document model into SDK create parameters.
func mapDocument(document *OnePasswordResourceItemDocumentModel) (*onepassword.DocumentCreateParams, error) {
	if document.Name.IsNull() || document.Name.IsUnknown() {
		return nil, fmt.Errorf("document name is required")
	}
	if document.Content.IsNull() || document.Content.IsUnknown() {
		return nil, fmt.Errorf("document content is required")
	}

	return &onepassword.DocumentCreateParams{
		Name:    document.Name.ValueString(),
		Content: []byte(document.Content.ValueString()),
	}, nil
}

// mapFile converts a file model into SDK create parameters tied to a field/section.
func mapFile(file *OnePasswordResourceItemFileModel, fieldID string, sectionID string) (*onepassword.FileCreateParams, error) {
	if file.Name.IsNull() || file.Name.IsUnknown() {
		return nil, fmt.Errorf("file name is required")
	}
	if file.Content.IsNull() || file.Content.IsUnknown() {
		return nil, fmt.Errorf("file content is required")
	}

	return &onepassword.FileCreateParams{
		Name:      file.Name.ValueString(),
		Content:   []byte(file.Content.ValueString()),
		SectionID: sectionID,
		FieldID:   fieldID,
	}, nil
}

// mapTags flattens the Terraform tag list into plain strings.
func mapTags(tags []types.String) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag.IsNull() || tag.IsUnknown() {
			continue
		}
		result = append(result, tag.ValueString())
	}
	return result
}

// getOptionalString reads a string value that may be null/unknown.
func getOptionalString(value types.String) (string, bool) {
	if value.IsNull() || value.IsUnknown() {
		return "", false
	}
	return value.ValueString(), true
}

// valueOrEmpty returns a string or empty when unknown/null.
func valueOrEmpty(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

// convertFilesToItemFiles maps file create parameters into item file metadata.
func convertFilesToItemFiles(files []onepassword.FileCreateParams) []onepassword.ItemFile {
	if len(files) == 0 {
		return nil
	}

	result := make([]onepassword.ItemFile, 0, len(files))
	for _, file := range files {
		result = append(result, onepassword.ItemFile{
			Attributes: onepassword.FileAttributes{
				Name: file.Name,
			},
			SectionID: file.SectionID,
			FieldID:   file.FieldID,
		})
	}

	return result
}
