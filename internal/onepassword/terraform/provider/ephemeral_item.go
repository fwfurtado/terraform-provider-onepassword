package onepasswordprovider

import (
	"context"
	"fmt"

	"github.com/1password/onepassword-sdk-go"
	"github.com/davecgh/go-spew/spew"
	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/client"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/samber/lo"
)

type OnePasswordItemModel struct {
	Vault types.String                `tfsdk:"vault"`
	Name  types.String                `tfsdk:"name"`
	Item  *OnePasswordSharedItemModel `tfsdk:"item"`
}

type OnePasswordEphemeralItem struct {
	client *client.ClientWrapper
}

// Configure implements [ephemeral.EphemeralResourceWithConfigure].
func (o *OnePasswordEphemeralItem) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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
func (o *OnePasswordEphemeralItem) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_item"
}

// Open implements [ephemeral.EphemeralResourceWithConfigure].
func (o *OnePasswordEphemeralItem) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data OnePasswordItemModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	vaultID, err := o.client.GetVault(ctx, data.Vault.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to get vault", err.Error())
		return
	}

	overview, err := o.client.GetItemOverview(ctx, vaultID, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("failed to get item item ID", err.Error())
		return
	}

	item, err := o.client.GetItem(ctx, overview.VaultID, overview.ID)
	if err != nil {
		resp.Diagnostics.AddError("failed to get item", err.Error())
		return
	}

	sectionFields := lo.GroupBy(item.Fields, func(field onepassword.ItemField) string {
		if field.SectionID == nil {
			return "no-section"
		}

		return *field.SectionID
	})

	sections := lo.MapToSlice(sectionFields, func(sectionID string, fields []onepassword.ItemField) *OnePasswordSharedItemSectionModel {
		title := "No Section"

		section, found := lo.Find(item.Sections, func(section onepassword.ItemSection) bool {
			return section.ID == sectionID
		})

		if found {
			title = section.Title
		}

		return &OnePasswordSharedItemSectionModel{
			ID:     types.StringValue(sectionID),
			Title:  types.StringValue(title),
			Fields: lo.Map(fields, mapField),
		}
	})

	data.Item = &OnePasswordSharedItemModel{
		ID:       types.StringValue(item.ID),
		Title:    types.StringValue(item.Title),
		Category: types.StringValue(string(item.Category)),
		Notes:    types.StringValue(item.Notes),
		Tags: lo.Map(item.Tags, func(tag string, _ int) types.String {
			return types.StringValue(tag)
		}),
		Websites: lo.Map(item.Websites, mapWebsite),
		Sections: sections,
		Version:  types.Int64Value(int64(item.Version)),
	}

	spew.Dump(data.Item)

	tflog.Debug(ctx, "1password: item", map[string]any{
		"item": data.Item,
	})

	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

// Schema implements [ephemeral.EphemeralResourceWithConfigure].
func (o *OnePasswordEphemeralItem) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "1Password item",
		Attributes: map[string]schema.Attribute{
			"vault": schema.StringAttribute{
				MarkdownDescription: "1Password vault title",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "1Password item name",
				Required:            true,
			},
			"item": schema.SingleNestedAttribute{
				MarkdownDescription: "1Password item data",
				Computed:            true,
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Item ID",
						Computed:            true,
					},
					"title": schema.StringAttribute{
						MarkdownDescription: "Item title",
						Computed:            true,
					},
					"category": schema.StringAttribute{
						MarkdownDescription: "Item category",
						Computed:            true,
					},
					"notes": schema.StringAttribute{
						MarkdownDescription: "Item notes",
						Computed:            true,
					},
					"version": schema.Int64Attribute{
						MarkdownDescription: "Item version",
						Computed:            true,
					},
					"tags": schema.ListAttribute{
						MarkdownDescription: "Item tags",
						Computed:            true,
						ElementType:         types.StringType,
					},
					"websites": schema.ListNestedAttribute{
						MarkdownDescription: "Item websites",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"url": schema.StringAttribute{
									MarkdownDescription: "Website URL",
									Computed:            true,
								},
								"label": schema.StringAttribute{
									MarkdownDescription: "Website label",
									Computed:            true,
								},
								"autofill_behavior": schema.StringAttribute{
									MarkdownDescription: "Website autofill behavior",
									Computed:            true,
								},
							},
						},
					},
					"sections": schema.ListNestedAttribute{
						MarkdownDescription: "Item sections",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									MarkdownDescription: "Section ID",
									Computed:            true,
								},
								"title": schema.StringAttribute{
									MarkdownDescription: "Section title",
									Computed:            true,
								},
								"fields": schema.ListNestedAttribute{
									MarkdownDescription: "Section fields",
									Computed:            true,
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"id": schema.StringAttribute{
												MarkdownDescription: "Field ID",
												Computed:            true,
											},
											"title": schema.StringAttribute{
												MarkdownDescription: "Field title",
												Computed:            true,
											},
											"field_type": schema.StringAttribute{
												MarkdownDescription: "Field type",
												Computed:            true,
											},
											"value": schema.StringAttribute{
												MarkdownDescription: "Field value",
												Computed:            true,
											},
											"file": schema.SingleNestedAttribute{
												MarkdownDescription: "Field file",
												Optional:            true,
												Attributes: map[string]schema.Attribute{
													"id": schema.StringAttribute{
														MarkdownDescription: "File ID",
														Computed:            true,
													},
													"name": schema.StringAttribute{
														MarkdownDescription: "File name",
														Computed:            true,
													},
												},
											},
										},
									},
								},
							},
						},
					},
					"document": schema.SingleNestedAttribute{
						MarkdownDescription: "Item document",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"id": schema.StringAttribute{
								MarkdownDescription: "Document ID",
								Computed:            true,
							},
							"name": schema.StringAttribute{
								MarkdownDescription: "Document name",
								Computed:            true,
							},
						},
					},
				},
			},
		},
	}
}

var (
	_ ephemeral.EphemeralResourceWithConfigure = &OnePasswordEphemeralItem{}
)

func mapWebsite(website onepassword.Website, _ int) *OnePasswordSharedItemWebsiteModel {
	return &OnePasswordSharedItemWebsiteModel{
		URL:              types.StringValue(website.URL),
		Label:            types.StringValue(website.Label),
		AutofillBehavior: types.StringValue(string(website.AutofillBehavior)),
	}
}

func mapField(field onepassword.ItemField, _ int) *OnePasswordSharedItemFieldModel {
	return &OnePasswordSharedItemFieldModel{
		ID:        types.StringValue(field.ID),
		Title:     types.StringValue(field.Title),
		FieldType: types.StringValue(string(field.FieldType)),
		Value:     types.StringValue(field.Value),
	}
}
