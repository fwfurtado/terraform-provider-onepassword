package onepasswordprovider

import (
	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type OnePasswordResourceItemModel struct {
	Vault types.String `tfsdk:"vault"`
	Title types.String `tfsdk:"title"`

	Category types.String `tfsdk:"category"`

	Database types.String `tfsdk:"database"`
	Port     types.Number `tfsdk:"port"`

	Hostname types.String `tfsdk:"hostname"`

	NoteValue          types.String `tfsdk:"note_value"`
	NoteValueWo        types.String `tfsdk:"note_value_wo"`
	NoteValueWoVersion types.Number `tfsdk:"note_value_wo_version"`

	Password          types.String                               `tfsdk:"password"`
	PasswordRecipe    OnePasswordResourceItemPasswordRecipeModel `tfsdk:"password_recipe"`
	PasswordWo        types.String                               `tfsdk:"password_wo"`
	PasswordWoVersion types.Number                               `tfsdk:"password_wo_version"`

	Section    OnePasswordResourceItemSectionModel `tfsdk:"section"`
	SectionMap types.MapType                       `tfsdk:"section_map"`
}

type OnePasswordResourceItemPasswordRecipeModel struct {
	Digits  types.Bool   `tfsdk:"digits"`
	Length  types.Number `tfsdk:"length"`
	Symbols types.Bool   `tfsdk:"symbols"`
}

type OnePasswordResourceItemSectionModel struct {
	ID    types.String                      `tfsdk:"id"`
	Label types.String                      `tfsdk:"label"`
	Field OnePasswordResourceItemFieldModel `tfsdk:"field"`
}

type OnePasswordResourceItemFieldModel struct {
	ID             types.String                               `tfsdk:"id"`
	Label          types.String                               `tfsdk:"label"`
	PasswordRecipe OnePasswordResourceItemPasswordRecipeModel `tfsdk:"password_recipe"`
	Type           types.String                               `tfsdk:"type"`
	Value          types.String                               `tfsdk:"value"`
}

type OnePasswordResourceItem struct {
	client *client.ClientWrapper
}
