# Example create a simple login item
resource "onepassword_login" "login" {
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name     = "Grafana 2"
  username = "grafana"
  password {
    random {
      length  = 32
      digits  = true
      symbols = true
    }
  }

  websites = {
    "website" = {
      url               = "https://grafana.com"
      autofill_behavior = "anywhere-on-website" # "anywhere-on-website", "exact-domain", "never"
    }
  }

  notes = "This is a note"
}


# Example create a secure note
resource "onepassword_secure_note" "secure_note" {
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name  = "Example Secure Note"
  notes = <<EOF
Example secure note

Without section
EOF
}

# Example create a credit card
resource "onepassword_credit_card" "credit_card" {
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name = "Example Credit Card"

  cardholder = "Ada Lovelace"
  number     = "4111111111111111" # FIXME: This don't work, the generated item in 1password does not show the card number
  type       = "visa"             # "visa", "mastercard", "american-express", "diners-club", "carte-blanche", "discover", "jcb", "maestro", "visa-electron", "laser", "unionpay"
  expiry     = "01/2030"
}

# Example create a password
resource "onepassword_password" "password" {
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name = "Example Password"
  password {
    memorable {
      word_count     = 4
      word_list_type = "full-words" # "full-words", "syllables", "three-letters"
      capitalize     = true
      separator_type = "hyphens" # "digits", "digits-and-symbols", "spaces", "hyphens", "underscores", "periods", "commas"
    }
  }

  # Optional
  websites = {
    "website" = {
      url               = "https://example.com"
      autofill_behavior = "anywhere-on-website" # "anywhere-on-website", "exact-domain", "never"
    }
  }
}

ephemeral "onepassword_secret" "password" {
    depends_on = [ onepassword_password.password ]
    reference = "op://${local.vault}/Example Password/password"
}


# Example create a document item
resource "onepassword_item" "document_item" {
  vault    = local.vault
  name     = "Example Document"
  category = "document"

  document {
    name    = "Example document"
    content = file("${path.module}/example.txt")
  }


  sections = {
    "metadata" = {
      label = "Metadata"
      fields = [
        {
          label = "owner"
          type  = "text"
          value = "Example Owner"
        }
      ]
    }
  }
}

# Example create a api credentials
resource "onepassword_api_credentials" "api_credentials" {
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name = "Example API Credentials"

  username = "api_user"
  credential {
    random {
      length  = 40
      symbols = false
      digits  = true
    }
  }
  type       = "bearer-token" #"bearer-token", "json-web-token", "json-credentials", "other" # FIXME: This don't work, field was saved with text instead of enum
  filename   = "api_credentials.json"
  valid_from = "2025-01-01" # FIXME: This don't work, field was saved with text instead of date
  expires    = "2026-01-01" # FIXME: This don't work, field was saved with text instead of date
  hostname   = "api.example.com"
}

# Example create a database
resource "onepassword_database" "database" {
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name = "Example Database"

  type               = "postgresql" # "ms-sql", "mysql", "oracle", "postgresql", "sqlite", "mongodb", "redis", "other"
  server             = "db.example.com"
  port               = 5432
  database           = "app"
  username           = "dbuser"
  sid                = "app"
  alias              = "app"
  connection_options = "SSL=true"

  password {
    random {
      length  = 40
      symbols = false
      digits  = true
    }
  }


}

# Example create a router
resource "onepassword_router" "router" {
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name = "Example Router"

  server_ip  = "192.168.1.1"
  airport_id = "1234567890"

  network_name = "Example Network"

  attached_storage_password {
    random {
      length  = 40
      symbols = false
      digits  = true
    }
  }

  base_station { # FIXME: This don't work, this blocx was not saved
    name = "router.local"
    password {
      random {
        length  = 40
        symbols = false
        digits  = true
      }
    }
  }

  wireless {               # FIXME: This don't work, this blocx was not saved
    security_type = "wpa2" # "wpa2", "wpa2-enterprise", "wpa3", "wpa3-enterprise", "wep", "wpa", "none"
    passphrase {
      random {
        length  = 40
        symbols = false
        digits  = true
      }
    }
  }
}

# Example create a server item

resource "onepassword_server" "server" {
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name = "Example Server"

  url      = "https://server.example.com"
  username = "root"
  password {
    random {
      length  = 40
      symbols = false
      digits  = true
    }
  }

  admin_console {
    url      = "https://server.example.com/admin"
    username = "admin"
    password {
      memorable {
        word_count     = 4
        word_list_type = "full-words" # "full-words", "syllables", "three-letters"
        capitalize     = true
        separator_type = "hyphens" # "digits", "digits-and-symbols", "spaces", "hyphens", "underscores", "periods", "commas"
      }
    }
  }

  hosting_provider {
    name    = "Example Hosting Provider"
    website = "https://example.com"
    support {
      url   = "https://example.com/support"
      email = "support@example.com"
      phone = "123-456-7890"
    }
  }
}

# Example create a ssh key
resource "onepassword_ssh_key" "generated_ssh_key" { # TODO: This resource must return the public key and fingerprint
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name = "Example Generated SSH Key"

  private_key {
    type      = "ed25519" # "rsa", "ed25519"
    generated = true
  }
}

resource "onepassword_ssh_key" "from_file_ssh_key" { # TODO: This resource must return the public key and fingerprint
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name = "Example Attached SSH Key"

  private_key {
    content   = file("${path.module}/key.pem") # This is a private key in PEM format
    generated = false                          # default is false
  }
}

# Example create a software license
resource "onepassword_software_license" "software_license" {
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name = "Example Software License"

  software {
    version     = "1.0.0"
    license_key = "AAAA-BBBB-CCCC-DDDD"
  }

  customer {
    licensed_to      = "Example Customer"
    registered_email = "customer@example.com"
    company          = "Example Company"
  }

  publisher {
    name          = "Example Publisher"
    donwload_page = "https://example.com/download"
    website       = "https://example.com"
    retail_price  = 100.00
    support_email = "support@example.com"
  }

  order {
    number        = "1234567890"
    total         = 100.00
    purchase_date = "2025-01-01"
  }

}