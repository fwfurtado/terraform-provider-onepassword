package onepasswordprovider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type OnePasswordEphemeralSecretModel struct {
	reference types.String `tfsdk:"reference"`
	Value     types.String `tfsdk:"value"`
}

type OnePasswordEphemeralSecret struct {
	client *client.ClientWrapper
}

var (
	_ ephemeral.EphemeralResourceWithConfigure = &OnePasswordEphemeralSecret{}
)

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

func (r *OnePasswordEphemeralSecret) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *OnePasswordEphemeralSecret) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get a secret from 1Password",
		Attributes: map[string]schema.Attribute{
			"reference": schema.StringAttribute{
				MarkdownDescription: "The secret reference to get the secret from",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^op://[a-zA-Z0-9_-]+/([a-zA-Z0-9_-]+/)+[a-zA-Z0-9_-]+$`), "Invalid secret reference"),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The returned secret value",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

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
			"secret_reference": data.reference.ValueString(),
		},
	)

	secret, err := r.client.GetSecretByReference(ctx, data.reference.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			"1password: failed to retrieve get item",
			fmt.Sprintf("Could not retrieve secret '%s': %s", data.reference.ValueString(), err.Error()),
		)
		return
	}

	data.Value = types.StringValue(secret)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)

	tflog.Debug(ctx, "1password: succesfully opened ephemeral resource")
}
