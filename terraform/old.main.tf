resource "onepassword_login" "login1" {
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

resource "onepassword_login" "login2" {
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name     = "Grafana 2"
  username = "grafana"

  password {
    pin {
      length = 6
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


resource "onepassword_login" "login3" {
  vault   = local.vault
  version = local.version
  tags    = local.tags

  name     = "Grafana 3"
  username = "grafana"

  password {
    memorable {
      word_count     = 8
      word_list_type = "syllables"
      capitalize     = true
      separator_type = "hyphens"
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