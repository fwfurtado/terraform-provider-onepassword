package onepasswordprovider

import "github.com/hashicorp/terraform-plugin-framework/types"

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
