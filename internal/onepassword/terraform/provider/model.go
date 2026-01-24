package onepasswordprovider

import "github.com/hashicorp/terraform-plugin-framework/types"

// OnePasswordSharedItemModel is the shared shape returned by item-based ephemerals.
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

// OnePasswordSharedItemSectionModel represents a section in a shared item.
type OnePasswordSharedItemSectionModel struct {
	ID     types.String                       `tfsdk:"id"`
	Title  types.String                       `tfsdk:"title"`
	Fields []*OnePasswordSharedItemFieldModel `tfsdk:"fields"`
}

// OnePasswordSharedItemFieldModel represents a field in a shared item section.
type OnePasswordSharedItemFieldModel struct {
	ID        types.String `tfsdk:"id"`
	Title     types.String `tfsdk:"title"`
	FieldType types.String `tfsdk:"field_type"`
	Value     types.String `tfsdk:"value"`
}

// OnePasswordSharedItemWebsiteModel represents a website entry on a shared item.
type OnePasswordSharedItemWebsiteModel struct {
	URL              types.String `tfsdk:"url"`
	Label            types.String `tfsdk:"label"`
	AutofillBehavior types.String `tfsdk:"autofill_behavior"`
}

// OnePasswordSharedPasswordModel represents a password in a shared item.
type OnePasswordSharedPasswordModel struct {
	Random    *OnePasswordSharedPasswordRandomModel    `tfsdk:"random"`
	Pin       *OnePasswordSharedPasswordPinModel       `tfsdk:"pin"`
	Memorable *OnePasswordSharedPasswordMemorableModel `tfsdk:"memorable"`
}

type OnePasswordSharedPasswordRandomModel struct {
	Length  types.Int64 `tfsdk:"length"`
	Digits  types.Bool  `tfsdk:"digits"`
	Symbols types.Bool  `tfsdk:"symbols"`
}

type OnePasswordSharedPasswordPinModel struct {
	Length types.Int64 `tfsdk:"length"`
}

type OnePasswordSharedPasswordMemorableModel struct {
	WordCount     types.Int64  `tfsdk:"word_count"`
	WordListType  types.String `tfsdk:"word_list_type"`
	Capitalize    types.Bool   `tfsdk:"capitalize"`
	SeparatorType types.String `tfsdk:"separator_type"`
}
