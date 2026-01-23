package onepasswordprovider

import (
	"context"
	"fmt"

	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/client"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type OnePasswordEphemeralItemModel struct {
	VaultID types.String                  `tfsdk:"vault_id"`
	Name    types.String                  `tfsdk:"name"`
	Item    *OnePasswordItemOverviewModel `tfsdk:"item"`
}

type OnePasswordItemOverviewModel struct {
	ID       types.String `tfsdk:"id"`
	Title    types.String `tfsdk:"title"`
	Category types.String `tfsdk:"category"`
}

type OnePasswordEphemeralItemOverview struct {
	client *client.ClientWrapper
}

// Configure implements [ephemeral.EphemeralResourceWithConfigure].
func (o *OnePasswordEphemeralItemOverview) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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
func (o *OnePasswordEphemeralItemOverview) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_item_overview"
}

// Schema implements [ephemeral.EphemeralResourceWithConfigure].
func (o *OnePasswordEphemeralItemOverview) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "1Password vault",
		Attributes: map[string]schema.Attribute{
			"vault_id": schema.StringAttribute{
				MarkdownDescription: "1Password vault title",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "1Password item name",
				Required:            true,
			},

			"item": schema.SingleNestedAttribute{
				MarkdownDescription: "1Password item overview",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "1Password item ID",
						Computed:            true,
					},
					"title": schema.StringAttribute{
						MarkdownDescription: "1Password item title",
						Computed:            true,
					},
					"category": schema.StringAttribute{
						MarkdownDescription: "1Password item category",
						Computed:            true,
					},
				},
			},
		},
	}
}

// Open implements [ephemeral.EphemeralResourceWithConfigure].
func (o *OnePasswordEphemeralItemOverview) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {

	var data OnePasswordEphemeralItemModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	item, err := o.client.GetItemOverview(ctx, data.VaultID.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to get vault", err.Error())
		return
	}

	data.Item = &OnePasswordItemOverviewModel{
		ID:       types.StringValue(item.ID),
		Title:    types.StringValue(item.Title),
		Category: types.StringValue(string(item.Category)),
	}

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

var (
	_ ephemeral.EphemeralResourceWithConfigure = &OnePasswordEphemeralItemOverview{}
)
