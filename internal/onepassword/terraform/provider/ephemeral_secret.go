package onepasswordprovider

import (
	"context"
	"fmt"

	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/client"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// OnePasswordEphemeralSecretModel holds the secret reference and resolved value.
type OnePasswordEphemeralSecretModel struct {
	Reference types.String `tfsdk:"reference"`
	Value     types.String `tfsdk:"value"`
}

// OnePasswordEphemeralSecret resolves a secret reference at apply time.
type OnePasswordEphemeralSecret struct {
	client *client.ClientWrapper
}

var (
	_ ephemeral.EphemeralResourceWithConfigure = &OnePasswordEphemeralSecret{}
)

// Configure stores the configured 1Password client.
func (r *OnePasswordEphemeralSecret) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.ClientWrapper)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			fmt.Sprintf("Expected *ClientWrapper, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if client == nil {
		resp.Diagnostics.AddError(
			"Unexpected resource configure type",
			"The 1Password client is required but was not configured. Please report this issue to the provider developers.",
		)
		return
	}

	r.client = client
}

// Metadata sets the ephemeral resource type name.
func (r *OnePasswordEphemeralSecret) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

// Schema defines the schema for the onepassword_secret ephemeral resource.
func (r *OnePasswordEphemeralSecret) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get a secret from 1Password.\n\n" +
			"Example:\n" +
			"```hcl\n" +
			"ephemeral \"onepassword_secret\" \"db_password\" {\n" +
			"  reference = \"op://Engineering/Database/password\"\n" +
			"}\n" +
			"\n" +
			"output \"db_password\" {\n" +
			"  value     = ephemeral.onepassword_secret.db_password.value\n" +
			"  sensitive = true\n" +
			"}\n" +
			"```\n\n" +
			"Reference: https://developer.1password.com/docs/cli/secret-reference-syntax/",
		Attributes: map[string]schema.Attribute{
			"reference": schema.StringAttribute{
				MarkdownDescription: "The secret reference to get the secret from.",
				Required:            true,
				Validators: []validator.String{
					&OnePasswordSecretReferenceValidator{client: r.client},
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The returned secret value.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

// Open resolves the secret reference and sets the value.
func (r *OnePasswordEphemeralSecret) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data OnePasswordEphemeralSecretModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(
		ctx,
		"1password: fetching secret",
		map[string]any{
			"secret_reference": data.Reference.ValueString(),
		},
	)

	secret, err := r.client.GetSecretByReference(ctx, data.Reference.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"1password: failed to retrieve get item",
			fmt.Sprintf("Could not retrieve secret '%s': %s", data.Reference.ValueString(), err.Error()),
		)
		return
	}

	data.Value = types.StringValue(secret)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)

	tflog.Debug(ctx, "1password: succesfully opened ephemeral resource")
}

// OnePasswordSecretReferenceValidator validates 1Password secret reference strings.
type OnePasswordSecretReferenceValidator struct {
	client *client.ClientWrapper
}

// Description implements [validator.String].
func (o *OnePasswordSecretReferenceValidator) Description(context.Context) string {
	return "string must be a valid 1Password secret reference format: op://<vault>/<item>[/<section>]/<field-name>?attribute=<attribute-value>"
}

// MarkdownDescription implements [validator.String].
func (o *OnePasswordSecretReferenceValidator) MarkdownDescription(context.Context) string {
	return "string must be a valid 1Password secret reference format: `op://<vault>/<item>[/<section>]/<field-name>?attribute=<attribute-value>`. [More information](https://developer.1password.com/docs/cli/secret-reference-syntax/)"
}

// ValidateString implements [validator.String].
func (o *OnePasswordSecretReferenceValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	err := o.client.ValidateSecretReference(ctx, req.ConfigValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"1password: failed to validate secret reference",
			err.Error(),
		)
	}

	resp.Diagnostics.Append(resp.Diagnostics...)
}

var (
	_ validator.String = &OnePasswordSecretReferenceValidator{}
)
