package onepasswordprovider

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/crypto/ssh"
)

// OnePasswordEphemeralSSHKeyModel represents the SSH key generator input/output.
type OnePasswordEphemeralSSHKeyModel struct {
	Type        types.String `tfsdk:"type"`
	PrivateKey  types.String `tfsdk:"private_key"`
	PublicKey   types.String `tfsdk:"public_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
	KeyType     types.String `tfsdk:"key_type"`
}

// OnePasswordEphemeralSSHKey generates SSH keys.
type OnePasswordEphemeralSSHKey struct{}

var (
	_ ephemeral.EphemeralResource = &OnePasswordEphemeralSSHKey{}
)

// Metadata sets the ephemeral resource type name.
func (r *OnePasswordEphemeralSSHKey) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

// Schema defines the schema for the SSH key generator.
func (r *OnePasswordEphemeralSSHKey) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Generate SSH key material for 1Password items.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				MarkdownDescription: "Key type (rsa or ed25519).",
				Required:            true,
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "Generated private key in PEM format.",
				Computed:            true,
				Sensitive:           true,
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "Generated public key.",
				Computed:            true,
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "Public key fingerprint.",
				Computed:            true,
			},
			"key_type": schema.StringAttribute{
				MarkdownDescription: "Generated key type.",
				Computed:            true,
			},
		},
	}
}

// Open generates the SSH key material.
func (r *OnePasswordEphemeralSSHKey) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var plan OnePasswordEphemeralSSHKeyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keyType := strings.ToLower(plan.Type.ValueString())
	privateKey, publicKey, fingerprint, normalizedType, err := generateSSHKey(keyType)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate SSH key", err.Error())
		return
	}

	plan.PrivateKey = types.StringValue(privateKey)
	plan.PublicKey = types.StringValue(publicKey)
	plan.Fingerprint = types.StringValue(fingerprint)
	plan.KeyType = types.StringValue(normalizedType)

	resp.Diagnostics.Append(resp.Result.Set(ctx, &plan)...)
}

func generateSSHKey(keyType string) (string, string, string, string, error) {
	switch keyType {
	case "ed25519":
		_, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return "", "", "", "", err
		}
		return marshalSSHKey(privateKey)
	case "rsa":
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return "", "", "", "", err
		}
		return marshalSSHKey(privateKey)
	default:
		return "", "", "", "", fmt.Errorf("unsupported key type %q", keyType)
	}
}

func marshalSSHKey(privateKey any) (string, string, string, string, error) {
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return "", "", "", "", err
	}

	block, err := ssh.MarshalPrivateKey(privateKey, "generated")
	if err != nil {
		return "", "", "", "", err
	}

	privateKeyPEM := strings.TrimSpace(string(pem.EncodeToMemory(block)))
	publicKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
	fingerprint := ssh.FingerprintSHA256(signer.PublicKey())

	return privateKeyPEM, publicKey, fingerprint, signer.PublicKey().Type(), nil
}
