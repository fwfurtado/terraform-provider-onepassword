## 1Password Terraform Provider

Terraform provider for managing 1Password items and fetching secrets using the
1Password SDK.

## Requirements

- Go (for local development)
- Terraform or OpenTofu
- 1Password Service Account token **or** 1Password Desktop App integration
- 1Password CLI if using desktop app integration

## Authentication

Exactly one of the following blocks must be configured:

### Service Account

```hcl
provider "onepassword" {
  service_account {
    token = var.service_account_token
  }
}
```

### Desktop App Integration

```hcl
provider "onepassword" {
  desktop_app_integration {
    account_name = "my.1password.com"
  }
}
```

## Default Tags

You can define `default_tags` at the provider level. These are merged with each
resource `tags` during create/update. When both are present, duplicates are
removed and the provider defaults are applied first.

```hcl
provider "onepassword" {
  service_account {
    token = var.service_account_token
  }

  default_tags {
    tags = {
      Environment = "Development"
      Project     = "OnePassword"
    }
  }
}
```

## Supported Resources

- `onepassword_login`
- `onepassword_secure_note`
- `onepassword_credit_card`
- `onepassword_password`
- `onepassword_document`
- `onepassword_api_credentials`
- `onepassword_database`
- `onepassword_router`
- `onepassword_server`
- `onepassword_ssh_key`
- `onepassword_software_license`

## Ephemeral Resources

- `ephemeral.onepassword_secret`

```hcl
ephemeral "onepassword_secret" "db_password" {
  reference = "op://Engineering/Database/password"
}
```

## Example Configuration

See `terraform/` for a full example, including a default provider setup and
resource configurations.

## Development

Useful targets:

- `make lint`
- `make test`
- `make run` (starts the provider in debug mode)
- `make apply`
- `make destroy`

The `apply` and `destroy` targets expect the provider to be running via
`make run` so they can read `TF_REATTACH_PROVIDERS` from `/tmp/provider.log`.
