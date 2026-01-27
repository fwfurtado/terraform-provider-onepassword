package onepasswordprovider

import (
	"context"
	"fmt"

	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// OnePasswordSecretDataSourceModel holds the secret reference and resolved value.
type OnePasswordSecretDataSourceModel struct {
	Reference types.String `tfsdk:"reference"`
	Value     types.String `tfsdk:"value"`
}

// OnePasswordSecretDataSource resolves a secret reference as a data source.
type OnePasswordSecretDataSource struct {
	client *client.ClientWrapper
}

var (
	_ datasource.DataSourceWithConfigure = &OnePasswordSecretDataSource{}
)

// Configure stores the configured 1Password client.
func (d *OnePasswordSecretDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*providerConfig)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected data source configure type",
			fmt.Sprintf("Expected *providerConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if config == nil || config.client == nil {
		resp.Diagnostics.AddError(
			"Unexpected data source configure type",
			"The 1Password client is required but was not configured. Please report this issue to the provider developers.",
		)
		return
	}

	d.client = config.client
}

// Metadata sets the data source type name.
func (d *OnePasswordSecretDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

// Schema defines the schema for the onepassword_secret data source.
func (d *OnePasswordSecretDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get a secret from 1Password.\n\n" +
			"Example:\n" +
			"```hcl\n" +
			"data \"onepassword_secret\" \"db_password\" {\n" +
			"  reference = \"op://Engineering/Database/password\"\n" +
			"}\n" +
			"\n" +
			"output \"db_password\" {\n" +
			"  value     = data.onepassword_secret.db_password.value\n" +
			"  sensitive = true\n" +
			"}\n" +
			"```\n\n" +
			"Reference: https://developer.1password.com/docs/cli/secret-reference-syntax/",
		Attributes: map[string]schema.Attribute{
			"reference": schema.StringAttribute{
				MarkdownDescription: "The secret reference to get the secret from.",
				Required:            true,
				Validators: []validator.String{
					&OnePasswordSecretReferenceValidator{client: d.client},
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The returned secret value. Warning: the value returned by this data source is persisted in Terraform state and should be used with care.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

// Read resolves the secret reference and sets the value.
func (d *OnePasswordSecretDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data OnePasswordSecretDataSourceModel
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

	secret, err := d.client.GetSecretByReference(ctx, data.Reference.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"1password: failed to retrieve get item",
			fmt.Sprintf("Could not retrieve secret '%s': %s", data.Reference.ValueString(), err.Error()),
		)
		return
	}

	data.Value = types.StringValue(secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Debug(ctx, "1password: succesfully read data source")
}
