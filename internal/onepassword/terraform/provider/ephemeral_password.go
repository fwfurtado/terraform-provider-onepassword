package onepasswordprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/1password/onepassword-sdk-go"
	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/client"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// OnePasswordEphemeralPasswordModel represents the input/output for password generation.
type OnePasswordEphemeralPasswordModel struct {
	Type          types.String `tfsdk:"type"`
	Length        types.Int64  `tfsdk:"length"`
	Digits        types.Bool   `tfsdk:"digits"`
	Symbols       types.Bool   `tfsdk:"symbols"`
	SeparatorType types.String `tfsdk:"separator_type"`
	Capitalize    types.Bool   `tfsdk:"capitalize"`
	WordListType  types.String `tfsdk:"word_list_type"`
	WordCount     types.Int64  `tfsdk:"word_count"`
	Password      types.String `tfsdk:"password"`
}

// OnePasswordEphemeralPassword generates passwords on demand.
type OnePasswordEphemeralPassword struct {
	client *client.ClientWrapper
}

var (
	_ ephemeral.EphemeralResourceWithConfigure = &OnePasswordEphemeralPassword{}
)

// Configure stores the configured 1Password client.
func (r *OnePasswordEphemeralPassword) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

// Metadata sets the ephemeral resource type name.
func (r *OnePasswordEphemeralPassword) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_password"
}

// Schema defines the schema for the onepassword_password ephemeral resource.
func (r *OnePasswordEphemeralPassword) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Generate a password using 1Password recipes.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				MarkdownDescription: "Password recipe type: random, pin, or memorable.",
				Required:            true,
			},
			"length": schema.Int64Attribute{
				MarkdownDescription: "Password length for random or pin recipes.",
				Optional:            true,
			},
			"digits": schema.BoolAttribute{
				MarkdownDescription: "Include digits in random recipes.",
				Optional:            true,
			},
			"symbols": schema.BoolAttribute{
				MarkdownDescription: "Include symbols in random recipes.",
				Optional:            true,
			},
			"separator_type": schema.StringAttribute{
				MarkdownDescription: "Separator type for memorable recipes.",
				Optional:            true,
			},
			"capitalize": schema.BoolAttribute{
				MarkdownDescription: "Capitalize one word in memorable recipes.",
				Optional:            true,
			},
			"word_list_type": schema.StringAttribute{
				MarkdownDescription: "Word list type for memorable recipes.",
				Optional:            true,
			},
			"word_count": schema.Int64Attribute{
				MarkdownDescription: "Word count for memorable recipes.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Generated password.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

// Open generates the password based on the provided recipe.
func (r *OnePasswordEphemeralPassword) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data OnePasswordEphemeralPasswordModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	recipe, err := buildPasswordRecipe(data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build password recipe", err.Error())
		return
	}

	password, err := r.client.GeneratePassword(ctx, recipe)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate password", err.Error())
		return
	}

	data.Password = types.StringValue(password)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

func buildPasswordRecipe(data OnePasswordEphemeralPasswordModel) (onepassword.PasswordRecipe, error) {
	if data.Type.IsNull() || data.Type.IsUnknown() {
		return onepassword.PasswordRecipe{}, fmt.Errorf("type is required")
	}

	switch strings.ToLower(data.Type.ValueString()) {
	case "random":
		length, err := requiredInt64(data.Length, "length")
		if err != nil {
			return onepassword.PasswordRecipe{}, err
		}

		return onepassword.NewPasswordRecipeTypeVariantRandom(&onepassword.PasswordRecipeRandomInner{
			IncludeDigits:  boolOrFalse(data.Digits),
			IncludeSymbols: boolOrFalse(data.Symbols),
			Length:         uint32(length),
		}), nil
	case "pin":
		length, err := requiredInt64(data.Length, "length")
		if err != nil {
			return onepassword.PasswordRecipe{}, err
		}

		return onepassword.NewPasswordRecipeTypeVariantPin(&onepassword.PasswordRecipePinInner{
			Length: uint32(length),
		}), nil
	case "memorable":
		wordCount, err := requiredInt64(data.WordCount, "word_count")
		if err != nil {
			return onepassword.PasswordRecipe{}, err
		}

		separator, err := requiredSeparatorType(data.SeparatorType)
		if err != nil {
			return onepassword.PasswordRecipe{}, err
		}

		wordList, err := requiredWordListType(data.WordListType)
		if err != nil {
			return onepassword.PasswordRecipe{}, err
		}

		return onepassword.NewPasswordRecipeTypeVariantMemorable(&onepassword.PasswordRecipeMemorableInner{
			SeparatorType: separator,
			Capitalize:    boolOrFalse(data.Capitalize),
			WordListType:  wordList,
			WordCount:     uint32(wordCount),
		}), nil
	default:
		return onepassword.PasswordRecipe{}, fmt.Errorf("unsupported recipe type: %s", data.Type.ValueString())
	}
}

func requiredInt64(value types.Int64, field string) (int64, error) {
	if value.IsNull() || value.IsUnknown() {
		return 0, fmt.Errorf("%s is required", field)
	}
	return value.ValueInt64(), nil
}

func boolOrFalse(value types.Bool) bool {
	if value.IsNull() || value.IsUnknown() {
		return false
	}
	return value.ValueBool()
}

func requiredSeparatorType(value types.String) (onepassword.SeparatorType, error) {
	if value.IsNull() || value.IsUnknown() {
		return "", fmt.Errorf("separator_type is required")
	}

	switch strings.ToLower(value.ValueString()) {
	case "digits":
		return onepassword.SeparatorTypeDigits, nil
	case "digits-and-symbols":
		return onepassword.SeparatorTypeDigitsAndSymbols, nil
	case "spaces":
		return onepassword.SeparatorTypeSpaces, nil
	case "hyphens":
		return onepassword.SeparatorTypeHyphens, nil
	case "underscores":
		return onepassword.SeparatorTypeUnderscores, nil
	case "periods":
		return onepassword.SeparatorTypePeriods, nil
	case "commas":
		return onepassword.SeparatorTypeCommas, nil
	default:
		return "", fmt.Errorf("unsupported separator type: %s", value.ValueString())
	}
}

func requiredWordListType(value types.String) (onepassword.WordListType, error) {
	if value.IsNull() || value.IsUnknown() {
		return "", fmt.Errorf("word_list_type is required")
	}

	switch strings.ToLower(value.ValueString()) {
	case "full-words":
		return onepassword.WordListTypeFullWords, nil
	case "syllables":
		return onepassword.WordListTypeSyllables, nil
	case "three-letters":
		return onepassword.WordListTypeThreeLetters, nil
	default:
		return "", fmt.Errorf("unsupported word list type: %s", value.ValueString())
	}
}
