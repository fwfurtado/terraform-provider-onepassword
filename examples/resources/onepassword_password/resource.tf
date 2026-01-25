resource "onepassword_password" "example" {
  vault   = "Engineering"
  version = 1
  tags    = ["terraform", "example"]

  name = "Example Password"
  password {
    memorable {
      word_count     = 4
      word_list_type = "full-words"
      capitalize     = true
      separator_type = "hyphens"
    }
  }

  websites = {
    "website" = {
      url               = "https://example.com"
      autofill_behavior = "anywhere-on-website"
    }
  }
}
