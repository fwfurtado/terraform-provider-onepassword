package onepasswordprovider

import (
	"context"
	"os"

	"github.com/fwfurtado/onepassword-tf-provider/internal/onepassword/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfprovides "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type OnePasswordProvider struct {
	version string
}

type OnePasswordProviderData struct {
	ServiceAccount        *OnePasswordProviderServiceAccountData        `tfsdk:"service_account"`
	DesktopAppIntegration *OnePasswordProviderDesktopAppIntegrationData `tfsdk:"desktop_app_integration"`
}

type OnePasswordProviderServiceAccountData struct {
	Token types.String `tfsdk:"token"`
}

type OnePasswordProviderDesktopAppIntegrationData struct {
	AccountName types.String `tfsdk:"account_name"`
}

func New(version string) *OnePasswordProvider {
	return &OnePasswordProvider{
		version: version,
	}
}

func (p *OnePasswordProvider) Metadata(_ context.Context, _ tfprovides.MetadataRequest, resp *tfprovides.MetadataResponse) {
	resp.TypeName = "onepassword"
	resp.Version = p.version
}

func (p *OnePasswordProvider) Schema(_ context.Context, _ tfprovides.SchemaRequest, resp *tfprovides.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The 1Password provider allows Terraform to retrieve secrets from 1Password using ephemeral resources. " +
			"It supports both Service Account authentication and Desktop App integration.",
		Attributes: map[string]schema.Attribute{

			"service_account": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for 1Password Service Account authentication. " +
					"Mutually exclusive with desktop_app_integration.",
				Optional: true,
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(path.Expression(path.MatchRoot("desktop_app_integration"))),
				},
				Attributes: map[string]schema.Attribute{
					"token": schema.StringAttribute{
						MarkdownDescription: "1Password Service Account token. Can also be set via OP_SERVICE_ACCOUNT_TOKEN environment variable.",
						Optional:            true,
						Sensitive:           true,
					},
				},
			},

			"desktop_app_integration": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration for 1Password Desktop App integration. " +
					"Requires 1Password CLI to be installed and configured. " +
					"Mutually exclusive with service_account.",
				Optional: true,
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(path.Expression(path.MatchRoot("service_account"))),
				},
				Attributes: map[string]schema.Attribute{
					"account_name": schema.StringAttribute{
						MarkdownDescription: "The 1Password account name to use. Optional - if not specified, uses the default account.",
						Optional:            true,
					},
				},
			},
		},
	}
}

func (p *OnePasswordProvider) Configure(ctx context.Context, req tfprovides.ConfigureRequest, resp *tfprovides.ConfigureResponse) {
	var data OnePasswordProviderData

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.ServiceAccount != nil && data.DesktopAppIntegration != nil {
		resp.Diagnostics.AddError(
			"Conflicting Authentication Methods",
			"Both service_account and desktop_app_integration are configured. Please specify only one authentication method.",
		)
		return
	}

	if data.ServiceAccount == nil && data.DesktopAppIntegration == nil {
		resp.Diagnostics.AddError(
			"Missing Authentication Configuration",
			"Either service_account or desktop_app_integration must be configured.",
		)
		return
	}

	var onePasswordClient *client.ClientWrapper

	if data.ServiceAccount != nil {
		token := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")

		if !data.ServiceAccount.Token.IsNull() && data.ServiceAccount.Token.ValueString() != "" {
			token = data.ServiceAccount.Token.ValueString()
		}

		if token == "" {
			resp.Diagnostics.AddError(
				"Missing Service Account Token",
				"The service_account.token is required but not set. "+
					"Provide it in the configuration or set the OP_SERVICE_ACCOUNT_TOKEN environment variable.",
			)

			return
		}

		tempOnePasswordClient, err := client.NewServiceAccount(ctx, p.version, token)

		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create 1Password Client",
				"Failed to create 1Password client with service account token.\n\n"+
					"Error: "+err.Error(),
			)
			return
		}

		onePasswordClient = tempOnePasswordClient
	}

	if data.DesktopAppIntegration != nil {
		accountName := os.Getenv("OP_ACCOUNT_NAME")

		if !data.DesktopAppIntegration.AccountName.IsNull() {
			accountName = data.DesktopAppIntegration.AccountName.ValueString()
		}

		if accountName == "" {
			resp.Diagnostics.AddError(
				"Missing Account Name",
				"The desktop_app_integration.account_name is required but not set. "+
					"Provide it in the configuration or set the OP_ACCOUNT_NAME environment variable.",
			)

			return
		}

		tempOnePasswordClient, err := client.NewDesktopAppIntegration(ctx, p.version, accountName)

		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create 1Password Client",
				"Failed to create 1Password client with desktop app integration.\n\n"+
					"Error: "+err.Error(),
			)
			return
		}

		onePasswordClient = tempOnePasswordClient

	}

	resp.ResourceData = onePasswordClient
	resp.DataSourceData = onePasswordClient
	resp.EphemeralResourceData = onePasswordClient
}

func (p *OnePasswordProvider) EphemeralResources(context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		func() ephemeral.EphemeralResource {
			return &OnePasswordEphemeralSecret{}
		},
		func() ephemeral.EphemeralResource {
			return &OnePasswordEphemeralItem{}
		},
		func() ephemeral.EphemeralResource {
			return &OnePasswordEphemeralPassword{}
		},
	}
}

func (p *OnePasswordProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *OnePasswordProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource {
			return &OnePasswordResourceItem{}
		},
	}
}

var (
	_ tfprovides.ProviderWithEphemeralResources = &OnePasswordProvider{}
)
