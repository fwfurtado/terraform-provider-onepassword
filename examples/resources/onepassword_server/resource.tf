resource "onepassword_server" "example" {
  vault   = "Engineering"
  version = 1
  tags    = ["terraform", "example"]

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
        word_list_type = "full-words"
        capitalize     = true
        separator_type = "hyphens"
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
