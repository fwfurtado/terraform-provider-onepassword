package onepasswordprovider

import "github.com/hashicorp/terraform-plugin-framework/types"

type OnePasswordSharedItemModel struct {
	ID       types.String                         `tfsdk:"id"`
	Title    types.String                         `tfsdk:"title"`
	Category types.String                         `tfsdk:"category"`
	Sections []*OnePasswordSharedItemSectionModel `tfsdk:"sections"`
	Notes    types.String                         `tfsdk:"notes"`
	Tags     []types.String                       `tfsdk:"tags"`
	Websites []*OnePasswordSharedItemWebsiteModel `tfsdk:"websites"`
	Version  types.Int64                          `tfsdk:"version"`
}

type OnePasswordSharedItemSectionModel struct {
	ID     types.String                       `tfsdk:"id"`
	Title  types.String                       `tfsdk:"title"`
	Fields []*OnePasswordSharedItemFieldModel `tfsdk:"fields"`
}

type OnePasswordSharedItemFieldModel struct {
	ID        types.String `tfsdk:"id"`
	Title     types.String `tfsdk:"title"`
	FieldType types.String `tfsdk:"field_type"`
	Value     types.String `tfsdk:"value"`
}

type OnePasswordSharedItemWebsiteModel struct {
	URL              types.String `tfsdk:"url"`
	Label            types.String `tfsdk:"label"`
	AutofillBehavior types.String `tfsdk:"autofill_behavior"`
}
