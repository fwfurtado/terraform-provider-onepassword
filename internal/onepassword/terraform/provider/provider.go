package onepasswordprovider

import (
	"context"
	"fmt"
	"os"
	"sort"

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

// OnePasswordProvider implements the Terraform provider for 1Password.
// It configures authentication and registers resources/ephemeral resources.
//
// References:
//   - https://developer.1password.com/docs/cli/
//   - https://developer.1password.com/docs/service-accounts/
type OnePasswordProvider struct {
	version string
}

// OnePasswordProviderData represents the provider configuration block.
type OnePasswordProviderData struct {
	ServiceAccount        *OnePasswordProviderServiceAccountData        `tfsdk:"service_account"`
	DesktopAppIntegration *OnePasswordProviderDesktopAppIntegrationData `tfsdk:"desktop_app_integration"`
	DefaultTags           *OnePasswordProviderDefaultTagsData           `tfsdk:"default_tags"`
}

// OnePasswordProviderServiceAccountData holds service account auth settings.
type OnePasswordProviderServiceAccountData struct {
	Token types.String `tfsdk:"token"`
}

// OnePasswordProviderDesktopAppIntegrationData holds desktop app integration settings.
type OnePasswordProviderDesktopAppIntegrationData struct {
	AccountName types.String `tfsdk:"account_name"`
}

type providerConfig struct {
	client      *client.ClientWrapper
	defaultTags []string
}

// OnePasswordProviderDefaultTagsData holds default item tags.
type OnePasswordProviderDefaultTagsData struct {
	Tags types.Map `tfsdk:"tags"`
}

// New returns a configured provider instance.
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
			"It supports both Service Account authentication and Desktop App integration.\n\n" +
			"Example (service account):\n" +
			"```hcl\n" +
			"provider \"onepassword\" {\n" +
			"  service_account {\n" +
			"    token = var.op_service_account_token\n" +
			"  }\n" +
			"}\n" +
			"```\n\n" +
			"Example (desktop app integration):\n" +
			"```hcl\n" +
			"provider \"onepassword\" {\n" +
			"  desktop_app_integration {\n" +
			"    account_name = \"my.1password.com\"\n" +
			"  }\n" +
			"}\n" +
			"```\n\n" +
			"References:\n" +
			"- https://developer.1password.com/docs/cli/\n" +
			"- https://developer.1password.com/docs/service-accounts/\n" +
			"- https://developer.hashicorp.com/terraform/language/providers/configuration",
		Blocks: map[string]schema.Block{

			"service_account": schema.SingleNestedBlock{
				MarkdownDescription: "Configuration for 1Password Service Account authentication. " +
					"Mutually exclusive with desktop_app_integration.\n\n" +
					"Reference: https://developer.1password.com/docs/service-accounts/",
				Validators: []validator.Object{
					objectvalidator.ExactlyOneOf(path.MatchRoot("desktop_app_integration")),
				},
				Attributes: map[string]schema.Attribute{
					"token": schema.StringAttribute{
						MarkdownDescription: "1Password Service Account token. Can also be set via OP_SERVICE_ACCOUNT_TOKEN environment variable.",
						Optional:            true,
						Sensitive:           true,
					},
				},
			},

			"desktop_app_integration": schema.SingleNestedBlock{
				MarkdownDescription: "Configuration for 1Password Desktop App integration. " +
					"Requires 1Password CLI to be installed and configured. " +
					"Mutually exclusive with service_account.\n\n" +
					"Reference: https://developer.1password.com/docs/cli/",
				Validators: []validator.Object{
					objectvalidator.ExactlyOneOf(path.MatchRoot("service_account")),
				},
				Attributes: map[string]schema.Attribute{
					"account_name": schema.StringAttribute{
						MarkdownDescription: "The 1Password account name to use. Optional - if not specified, uses the default account.",
						Optional:            true,
					},
				},
			},
			"default_tags": schema.SingleNestedBlock{
				MarkdownDescription: "Default tags to apply to all items created by this provider.",
				Attributes: map[string]schema.Attribute{
					"tags": schema.MapAttribute{
						MarkdownDescription: "Default tag key-value pairs.",
						Optional:            true,
						ElementType:         types.StringType,
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
	var defaultTags []string

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

	if data.DefaultTags != nil && !data.DefaultTags.Tags.IsNull() && !data.DefaultTags.Tags.IsUnknown() {
		tagMap := map[string]string{}
		resp.Diagnostics.Append(data.DefaultTags.Tags.ElementsAs(ctx, &tagMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		keys := make([]string, 0, len(tagMap))
		for key := range tagMap {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := tagMap[key]
			if value == "" {
				defaultTags = append(defaultTags, key)
				continue
			}
			defaultTags = append(defaultTags, fmt.Sprintf("%s=%s", key, value))
		}
	}

	config := &providerConfig{
		client:      onePasswordClient,
		defaultTags: defaultTags,
	}
	resp.ResourceData = config
	resp.DataSourceData = config
	resp.EphemeralResourceData = config
}

func (p *OnePasswordProvider) EphemeralResources(context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		func() ephemeral.EphemeralResource {
			return &OnePasswordEphemeralSecret{}
		},
	}
}

func (p *OnePasswordProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource {
			return &OnePasswordSecretDataSource{}
		},
	}
}

func (p *OnePasswordProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource {
			return &OnePasswordLogin{}
		},
		func() resource.Resource {
			return &OnePasswordSecureNote{}
		},
		func() resource.Resource {
			return &OnePasswordCreditCard{}
		},
		func() resource.Resource {
			return &OnePasswordPasswordItem{}
		},
		func() resource.Resource {
			return &OnePasswordDocument{}
		},
		func() resource.Resource {
			return &OnePasswordAPICredentials{}
		},
		func() resource.Resource {
			return &OnePasswordDatabase{}
		},
		func() resource.Resource {
			return &OnePasswordRouter{}
		},
		func() resource.Resource {
			return &OnePasswordServer{}
		},
		func() resource.Resource {
			return &OnePasswordSSHKey{}
		},
		func() resource.Resource {
			return &OnePasswordSoftwareLicense{}
		},
	}
}

var (
	_ tfprovides.ProviderWithEphemeralResources = &OnePasswordProvider{}
)
