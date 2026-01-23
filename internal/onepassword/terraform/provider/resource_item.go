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
	Notes           types.String                          `tfsdk:"notes"`
	Document        *OnePasswordResourceItemDocumentModel `tfsdk:"document"`
	Sections        []OnePasswordResourceItemSectionModel `tfsdk:"sections"`
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
	Label   types.String                         `tfsdk:"label"`
	Type    types.String                         `tfsdk:"type"`
	Value   types.String                         `tfsdk:"value"`
	File    *OnePasswordResourceItemFileModel    `tfsdk:"file"`
	Address *OnePasswordResourceItemAddressModel `tfsdk:"address"`
	SSHKey  *OnePasswordResourceItemSSHKeyModel  `tfsdk:"ssh_key"`
	TOTP    *OnePasswordResourceItemTOTPModel    `tfsdk:"totp"`
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
		MarkdownDescription: "Manage 1Password items.",
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
				MarkdownDescription: "Item category.",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username field value.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password field value.",
				Optional:            true,
				Sensitive:           true,
			},
			"password_version": schema.Int64Attribute{
				MarkdownDescription: "Version used to force password rotation.",
				Optional:            true,
			},
			"notes": schema.StringAttribute{
				MarkdownDescription: "Item notes.",
				Optional:            true,
			},
			"tags": schema.ListAttribute{
				MarkdownDescription: "Item tags.",
				Optional:            true,
				ElementType:         types.StringType,
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
					},
				},
			},
			"sections": schema.ListNestedAttribute{
				MarkdownDescription: "Item sections and fields.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label": schema.StringAttribute{
							MarkdownDescription: "Section label. Empty label places fields at root.",
							Optional:            true,
						},
						"fields": schema.ListNestedAttribute{
							MarkdownDescription: "Fields within the section.",
							Required:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"label": schema.StringAttribute{
										MarkdownDescription: "Field label.",
										Optional:            true,
									},
									"type": schema.StringAttribute{
										MarkdownDescription: "Field type.",
										Required:            true,
									},
									"value": schema.StringAttribute{
										MarkdownDescription: "Field value.",
										Optional:            true,
									},
									"file": schema.SingleNestedAttribute{
										MarkdownDescription: "File attachment for the field.",
										Optional:            true,
										Attributes: map[string]schema.Attribute{
											"name": schema.StringAttribute{
												MarkdownDescription: "File name.",
												Required:            true,
											},
											"content": schema.StringAttribute{
												MarkdownDescription: "File content.",
												Required:            true,
												Sensitive:           true,
											},
										},
									},
									"address": schema.SingleNestedAttribute{
										MarkdownDescription: "Address details for address fields.",
										Optional:            true,
										Attributes: map[string]schema.Attribute{
											"street": schema.StringAttribute{
												MarkdownDescription: "Street.",
												Optional:            true,
											},
											"city": schema.StringAttribute{
												MarkdownDescription: "City.",
												Optional:            true,
											},
											"state": schema.StringAttribute{
												MarkdownDescription: "State.",
												Optional:            true,
											},
											"zip": schema.StringAttribute{
												MarkdownDescription: "ZIP code.",
												Optional:            true,
											},
											"country": schema.StringAttribute{
												MarkdownDescription: "Country.",
												Optional:            true,
											},
										},
									},
									"ssh_key": schema.SingleNestedAttribute{
										MarkdownDescription: "SSH key details for SSH key fields.",
										Optional:            true,
										Attributes: map[string]schema.Attribute{
											"public_key": schema.StringAttribute{
												MarkdownDescription: "SSH public key.",
												Optional:            true,
											},
											"fingerprint": schema.StringAttribute{
												MarkdownDescription: "SSH key fingerprint.",
												Optional:            true,
											},
											"key_type": schema.StringAttribute{
												MarkdownDescription: "SSH key type.",
												Optional:            true,
											},
										},
									},
									"totp": schema.SingleNestedAttribute{
										MarkdownDescription: "TOTP details for TOTP fields.",
										Optional:            true,
										Attributes: map[string]schema.Attribute{
											"code": schema.StringAttribute{
												MarkdownDescription: "Computed TOTP code.",
												Optional:            true,
											},
											"error_message": schema.StringAttribute{
												MarkdownDescription: "TOTP error message.",
												Optional:            true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
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

	if !plan.Notes.IsNull() && !plan.Notes.IsUnknown() {
		notes := plan.Notes.ValueString()
		params.Notes = &notes
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

	if !plan.Notes.IsNull() && !plan.Notes.IsUnknown() {
		item.Notes = plan.Notes.ValueString()
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

func buildItemSectionsAndFields(plan *OnePasswordResourceItemModel, existing *onepassword.Item) ([]onepassword.ItemSection, []onepassword.ItemField, []onepassword.FileCreateParams, error) {
	sections := make([]onepassword.ItemSection, 0)
	fields := make([]onepassword.ItemField, 0)
	files := make([]onepassword.FileCreateParams, 0)

	sectionIDs := map[string]string{}
	fieldIDs := map[string]string{}

	if existing != nil {
		for _, section := range existing.Sections {
			sectionIDs[section.Title] = section.ID
		}

		for _, field := range existing.Fields {
			sectionKey := "root"
			if field.SectionID != nil {
				sectionKey = *field.SectionID
			}
			fieldIDs[sectionKey+":"+field.Title] = field.ID
		}
	}

	if username, ok := getOptionalString(plan.Username); ok {
		fieldID := fieldIDs["root:username"]
		if fieldID == "" {
			fieldID = "username"
		}
		fields = append(fields, onepassword.ItemField{
			ID:        fieldID,
			Title:     "username",
			FieldType: onepassword.ItemFieldTypeText,
			Value:     username,
		})
	}

	if password, ok := getOptionalString(plan.Password); ok {
		fieldID := fieldIDs["root:password"]
		if fieldID == "" {
			fieldID = "password"
		}
		fields = append(fields, onepassword.ItemField{
			ID:        fieldID,
			Title:     "password",
			FieldType: onepassword.ItemFieldTypeConcealed,
			Value:     password,
		})
	}

	for sectionIndex, section := range plan.Sections {
		sectionLabel, hasLabel := getOptionalString(section.Label)
		var sectionID string
		if hasLabel {
			sectionID = sectionIDs[sectionLabel]
			if sectionID == "" {
				sectionID = fmt.Sprintf("section-%d", sectionIndex)
			}
			sections = append(sections, onepassword.ItemSection{
				ID:    sectionID,
				Title: sectionLabel,
			})
		}

		for fieldIndex, fieldModel := range section.Fields {
			fieldType, err := mapFieldType(fieldModel.Type)
			if err != nil {
				return nil, nil, nil, err
			}

			fieldLabel, _ := getOptionalString(fieldModel.Label)
			fieldValue, _ := getOptionalString(fieldModel.Value)

			sectionKey := "root"
			var sectionIDPtr *string
			if hasLabel {
				sectionKey = sectionID
				sectionIDPtr = &sectionID
			}

			fieldID := fieldIDs[sectionKey+":"+fieldLabel]
			if fieldID == "" {
				fieldID = fmt.Sprintf("field-%d-%d", sectionIndex, fieldIndex)
			}

			itemField := onepassword.ItemField{
				ID:        fieldID,
				Title:     fieldLabel,
				SectionID: sectionIDPtr,
				FieldType: fieldType,
				Value:     fieldValue,
			}

			if fieldModel.Address != nil {
				address := onepassword.AddressFieldDetails{
					Street:  valueOrEmpty(fieldModel.Address.Street),
					City:    valueOrEmpty(fieldModel.Address.City),
					State:   valueOrEmpty(fieldModel.Address.State),
					Zip:     valueOrEmpty(fieldModel.Address.Zip),
					Country: valueOrEmpty(fieldModel.Address.Country),
				}
				details := onepassword.NewItemFieldDetailsTypeVariantAddress(&address)
				itemField.Details = &details
			}

			if fieldModel.SSHKey != nil {
				sshKey := onepassword.SSHKeyAttributes{
					PublicKey:   valueOrEmpty(fieldModel.SSHKey.PublicKey),
					Fingerprint: valueOrEmpty(fieldModel.SSHKey.Fingerprint),
					KeyType:     valueOrEmpty(fieldModel.SSHKey.KeyType),
				}
				details := onepassword.NewItemFieldDetailsTypeVariantSSHKey(&sshKey)
				itemField.Details = &details
			}

			if fieldModel.TOTP != nil {
				totp := onepassword.OTPFieldDetails{}
				if code, ok := getOptionalString(fieldModel.TOTP.Code); ok {
					totp.Code = &code
				}
				if message, ok := getOptionalString(fieldModel.TOTP.ErrorMessage); ok {
					totp.ErrorMessage = &message
				}
				details := onepassword.NewItemFieldDetailsTypeVariantOTP(&totp)
				itemField.Details = &details
			}

			fields = append(fields, itemField)

			if fieldModel.File != nil {
				file, err := mapFile(fieldModel.File, fieldID, sectionID)
				if err != nil {
					return nil, nil, nil, err
				}
				files = append(files, *file)
			}
		}
	}

	return sections, fields, files, nil
}

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

func getOptionalString(value types.String) (string, bool) {
	if value.IsNull() || value.IsUnknown() {
		return "", false
	}
	return value.ValueString(), true
}

func valueOrEmpty(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

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
