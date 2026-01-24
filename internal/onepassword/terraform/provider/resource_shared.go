package onepasswordprovider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/1password/onepassword-sdk-go"
	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/ssh"
)

// SharedItemModel holds the common fields for specialized 1Password resources.
type SharedItemModel struct {
	ID       types.String                `tfsdk:"id"`
	Vault    types.String                `tfsdk:"vault"`
	Name     types.String                `tfsdk:"name"`
	Note     types.String                `tfsdk:"note"`
	Notes    types.String                `tfsdk:"notes"`
	Tags     []types.String              `tfsdk:"tags"`
	Version  types.Int64                 `tfsdk:"version"`
	Sections *OnePasswordSectionMapModel `tfsdk:"sections"`
}

// OnePasswordSectionMapModel represents root fields and labeled sections.
type OnePasswordSectionMapModel struct {
	Default []OnePasswordResourceItemFieldModel       `tfsdk:"default"`
	Labeled map[string]OnePasswordLabeledSectionModel `tfsdk:"labeled"`
}

// OnePasswordLabeledSectionModel represents a labeled section.
type OnePasswordLabeledSectionModel struct {
	Label  types.String                        `tfsdk:"label"`
	Fields []OnePasswordResourceItemFieldModel `tfsdk:"fields"`
}

// OnePasswordWebsiteMapModel represents websites in map form.
type OnePasswordWebsiteMapModel struct {
	URL              types.String `tfsdk:"url"`
	AutofillBehavior types.String `tfsdk:"autofill_behavior"`
}

// BaseItemResource holds the configured client.
type BaseItemResource struct {
	client *client.ClientWrapper
}

// Configure stores the configured 1Password client.
func (r *BaseItemResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func sharedItemAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
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
		"notes": schema.StringAttribute{
			MarkdownDescription: "Item notes (deprecated in favor of note).",
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
		"sections": sectionsAttribute("Item sections map."),
	}
}

func sectionsAttribute(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: description,
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"default": schema.ListNestedAttribute{
				MarkdownDescription: "Root-level fields.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: fieldAttributes(),
				},
			},
			"labeled": schema.MapNestedAttribute{
				MarkdownDescription: "Labeled sections keyed by name.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"label": schema.StringAttribute{
							MarkdownDescription: "Section label.",
							Optional:            true,
						},
						"fields": schema.ListNestedAttribute{
							MarkdownDescription: "Fields in this section.",
							Required:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: fieldAttributes(),
							},
						},
					},
				},
			},
		},
	}
}

func fieldAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"label": schema.StringAttribute{
			MarkdownDescription: "Field label.",
			Optional:            true,
		},
		"id": schema.StringAttribute{
			MarkdownDescription: "Field ID.",
			Optional:            true,
		},
		"type": schema.StringAttribute{
			MarkdownDescription: "Field type.",
			Required:            true,
		},
		"value": schema.StringAttribute{
			MarkdownDescription: "Field value.",
			Optional:            true,
			WriteOnly:           true,
		},
		"metadata": schema.MapAttribute{
			MarkdownDescription: "Field metadata.",
			Optional:            true,
			ElementType:         types.StringType,
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
					WriteOnly:           true,
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
				"private_key": schema.StringAttribute{
					MarkdownDescription: "SSH private key.",
					Optional:            true,
					Sensitive:           true,
					WriteOnly:           true,
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
					WriteOnly:           true,
				},
				"error_message": schema.StringAttribute{
					MarkdownDescription: "TOTP error message.",
					Optional:            true,
				},
			},
		},
	}
}

func buildSectionsFromMap(
	sectionsModel *OnePasswordSectionMapModel,
	existing *onepassword.Item,
) ([]onepassword.ItemSection, []onepassword.ItemField, []onepassword.FileCreateParams, error) {
	sections := make([]onepassword.ItemSection, 0)
	fields := make([]onepassword.ItemField, 0)
	files := make([]onepassword.FileCreateParams, 0)

	if sectionsModel == nil {
		return sections, fields, files, nil
	}

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

	for index, fieldModel := range sectionsModel.Default {
		fieldLabel, fieldValue := resolveFieldLabelValue(fieldModel)
		fieldID := resolveFieldID(fieldModel, fieldLabel, "root", fieldIDs, index, "field-root")
		itemField, file, err := buildItemFieldFromModel(fieldModel, fieldID, fieldLabel, fieldValue, nil, "")
		if err != nil {
			return nil, nil, nil, err
		}
		fields = append(fields, itemField)
		if file != nil {
			files = append(files, *file)
		}
	}

	labeledKeys := make([]string, 0, len(sectionsModel.Labeled))
	for key := range sectionsModel.Labeled {
		labeledKeys = append(labeledKeys, key)
	}
	sort.Strings(labeledKeys)

	for sectionIndex, key := range labeledKeys {
		section := sectionsModel.Labeled[key]
		sectionLabel := valueOrEmpty(section.Label)
		if sectionLabel == "" {
			sectionLabel = key
		}

		sectionID := sectionIDs[sectionLabel]
		if sectionID == "" {
			sectionID = fmt.Sprintf("section-%d", sectionIndex)
		}
		sections = append(sections, onepassword.ItemSection{
			ID:    sectionID,
			Title: sectionLabel,
		})

		for fieldIndex, fieldModel := range section.Fields {
			fieldLabel, fieldValue := resolveFieldLabelValue(fieldModel)
			fieldID := resolveFieldID(fieldModel, fieldLabel, sectionID, fieldIDs, fieldIndex, "field")
			itemField, file, err := buildItemFieldFromModel(fieldModel, fieldID, fieldLabel, fieldValue, &sectionID, sectionID)
			if err != nil {
				return nil, nil, nil, err
			}
			fields = append(fields, itemField)
			if file != nil {
				files = append(files, *file)
			}
		}
	}

	return sections, fields, files, nil
}

func resolveFieldID(
	fieldModel OnePasswordResourceItemFieldModel,
	fieldLabel string,
	sectionKey string,
	fieldIDs map[string]string,
	index int,
	prefix string,
) string {
	if fieldID, ok := getOptionalString(fieldModel.ID); ok && fieldID != "" {
		return fieldID
	}

	if fieldLabel != "" {
		if existingID := fieldIDs[sectionKey+":"+fieldLabel]; existingID != "" {
			return existingID
		}
		return fieldLabel
	}

	return fmt.Sprintf("%s-%d", prefix, index)
}

func resolveFieldLabelValue(fieldModel OnePasswordResourceItemFieldModel) (string, string) {
	fieldLabel, _ := getOptionalString(fieldModel.Label)
	fieldValue, _ := getOptionalString(fieldModel.Value)

	if fieldModel.SSHKey != nil {
		if privateKey, ok := getOptionalString(fieldModel.SSHKey.PrivateKey); ok {
			fieldValue = privateKey
		}
	}

	return fieldLabel, fieldValue
}

func buildItemFieldFromModel(
	fieldModel OnePasswordResourceItemFieldModel,
	fieldID string,
	fieldLabel string,
	fieldValue string,
	sectionIDPtr *string,
	sectionID string,
) (onepassword.ItemField, *onepassword.FileCreateParams, error) {
	fieldType, err := mapFieldType(fieldModel.Type)
	if err != nil {
		return onepassword.ItemField{}, nil, err
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

	if fieldModel.File != nil {
		file, err := mapFile(fieldModel.File, fieldID, sectionID)
		if err != nil {
			return onepassword.ItemField{}, nil, err
		}
		return itemField, file, nil
	}

	return itemField, nil, nil
}

func noteFromShared(model SharedItemModel) *string {
	if value, ok := getOptionalString(model.Note); ok {
		return &value
	}
	if value, ok := getOptionalString(model.Notes); ok {
		return &value
	}
	return nil
}

func mapWebsitesFromMap(websites map[string]OnePasswordWebsiteMapModel) ([]onepassword.Website, error) {
	if len(websites) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(websites))
	for key := range websites {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]onepassword.Website, 0, len(websites))
	for _, key := range keys {
		website := websites[key]
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
			Label:            key,
			AutofillBehavior: autofill,
		})
	}

	return result, nil
}

func sharedFieldID(label string) string {
	normalized := strings.ToLower(strings.TrimSpace(label))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	if normalized == "" {
		return "field"
	}
	return normalized
}

type FieldInput struct {
	Label   string
	Type    onepassword.ItemFieldType
	Value   string
	Details *onepassword.ItemFieldDetails
	ID      string
}

func addStringField(inputs *[]FieldInput, label string, fieldType onepassword.ItemFieldType, value types.String) {
	if inputValue, ok := getOptionalString(value); ok {
		*inputs = append(*inputs, FieldInput{
			ID:    label,
			Label: label,
			Type:  fieldType,
			Value: inputValue,
		})
	}
}

func addConcealedField(inputs *[]FieldInput, label string, value types.String) {
	if inputValue, ok := getOptionalString(value); ok {
		*inputs = append(*inputs, FieldInput{
			ID:    label,
			Label: label,
			Type:  onepassword.ItemFieldTypeConcealed,
			Value: inputValue,
		})
	}
}

func addIntField(inputs *[]FieldInput, label string, fieldType onepassword.ItemFieldType, value types.Int64) {
	if value.IsNull() || value.IsUnknown() {
		return
	}
	*inputs = append(*inputs, FieldInput{
		ID:    label,
		Label: label,
		Type:  fieldType,
		Value: fmt.Sprintf("%d", value.ValueInt64()),
	})
}

func addNumberField(inputs *[]FieldInput, label string, fieldType onepassword.ItemFieldType, value types.Number) {
	if value.IsNull() || value.IsUnknown() {
		return
	}
	*inputs = append(*inputs, FieldInput{
		ID:    label,
		Label: label,
		Type:  fieldType,
		Value: value.ValueBigFloat().Text('f', -1),
	})
}

func buildFieldsFromInputs(inputs []FieldInput, existing *onepassword.Item) []onepassword.ItemField {
	if len(inputs) == 0 {
		return nil
	}

	fieldIDs := map[string]string{}
	if existing != nil {
		for _, field := range existing.Fields {
			sectionKey := "root"
			if field.SectionID != nil {
				sectionKey = *field.SectionID
			}
			fieldIDs[sectionKey+":"+field.Title] = field.ID
		}
	}

	fields := make([]onepassword.ItemField, 0, len(inputs))
	for _, input := range inputs {
		if input.Value == "" && input.Details == nil {
			continue
		}

		fieldID := input.ID
		if fieldID == "" && input.Label != "" {
			if existingID := fieldIDs["root:"+input.Label]; existingID != "" {
				fieldID = existingID
			} else {
				fieldID = sharedFieldID(input.Label)
			}
		}
		if fieldID == "" {
			fieldID = "field"
		}

		fields = append(fields, onepassword.ItemField{
			ID:        fieldID,
			Title:     input.Label,
			FieldType: input.Type,
			Value:     input.Value,
			Details:   input.Details,
		})
	}

	return fields
}

func buildItemCreateParamsFromShared(
	category onepassword.ItemCategory,
	vaultID string,
	base SharedItemModel,
	extraFields []onepassword.ItemField,
	websites []onepassword.Website,
	document *onepassword.DocumentCreateParams,
) (onepassword.ItemCreateParams, error) {
	sections, fields, files, err := buildSectionsFromMap(base.Sections, nil)
	if err != nil {
		return onepassword.ItemCreateParams{}, err
	}

	if len(extraFields) > 0 {
		fields = append(fields, extraFields...)
	}

	params := onepassword.ItemCreateParams{
		Category: category,
		VaultID:  vaultID,
		Title:    base.Name.ValueString(),
		Fields:   fields,
		Sections: sections,
		Files:    files,
	}

	if note := noteFromShared(base); note != nil {
		params.Notes = note
	}

	if len(base.Tags) > 0 {
		params.Tags = mapTags(base.Tags)
	}

	if len(websites) > 0 {
		params.Websites = websites
	}

	if document != nil {
		params.Document = document
	}

	return params, nil
}

func buildItemForUpdateFromShared(
	category onepassword.ItemCategory,
	vaultID string,
	base SharedItemModel,
	existing *onepassword.Item,
	extraFields []onepassword.ItemField,
	websites []onepassword.Website,
	document *onepassword.DocumentCreateParams,
) (onepassword.Item, error) {
	sections, fields, files, err := buildSectionsFromMap(base.Sections, existing)
	if err != nil {
		return onepassword.Item{}, err
	}

	if len(extraFields) > 0 {
		fields = append(fields, extraFields...)
	}

	item := onepassword.Item{
		ID:       existing.ID,
		VaultID:  vaultID,
		Title:    base.Name.ValueString(),
		Category: category,
		Fields:   fields,
		Sections: sections,
		Files:    convertFilesToItemFiles(files),
		Version:  existing.Version,
	}

	if note := noteFromShared(base); note != nil {
		item.Notes = *note
	}

	if len(base.Tags) > 0 {
		item.Tags = mapTags(base.Tags)
	}

	if len(websites) > 0 {
		item.Websites = websites
	}

	if document != nil {
		item.Document = &onepassword.FileAttributes{Name: document.Name}
	}

	return item, nil
}

func parseSSHKeyDetails(privateKey string) (string, string, string, error) {
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return "", "", "", err
	}

	publicKey := string(ssh.MarshalAuthorizedKey(signer.PublicKey()))
	fingerprint := ssh.FingerprintSHA256(signer.PublicKey())
	keyType := signer.PublicKey().Type()

	return strings.TrimSpace(publicKey), fingerprint, keyType, nil
}
