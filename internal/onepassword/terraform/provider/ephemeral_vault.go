package onepasswordprovider

import (
	"context"
	"fmt"

	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/client"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type OnePasswordEphemeralVaultModel struct {
	Vault types.String `tfsdk:"vault"`
	ID    types.String `tfsdk:"id"`
}

type OnePasswordEphemeralVault struct {
	client *client.ClientWrapper
}

// Configure implements [ephemeral.EphemeralResourceWithConfigure].
func (o *OnePasswordEphemeralVault) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

	o.client = client
}

// Metadata implements [ephemeral.EphemeralResourceWithConfigure].
func (o *OnePasswordEphemeralVault) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault"
}

// Schema implements [ephemeral.EphemeralResourceWithConfigure].
func (o *OnePasswordEphemeralVault) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "1Password vault",
		Attributes: map[string]schema.Attribute{
			"vault": schema.StringAttribute{
				MarkdownDescription: "1Password vault title",
				Required:            true,
			},

			"id": schema.StringAttribute{
				MarkdownDescription: "1Password vault ID",
				Computed:            true,
			},
		},
	}
}

// Open implements [ephemeral.EphemeralResourceWithConfigure].
func (o *OnePasswordEphemeralVault) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {

	var data OnePasswordEphemeralVaultModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := o.client.GetVault(ctx, data.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to get vault", err.Error())
		return
	}

	data.ID = types.StringValue(vaultID)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

var (
	_ ephemeral.EphemeralResourceWithConfigure = &OnePasswordEphemeralVault{}
)
