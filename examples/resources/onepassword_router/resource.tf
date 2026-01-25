resource "onepassword_router" "example" {
  vault   = "Engineering"
  version = 1
  tags    = ["terraform", "example"]

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

  base_station {
    name = "router.local"
    password {
      random {
        length  = 40
        symbols = false
        digits  = true
      }
    }
  }

  wireless {
    security_type = "wpa2"
    passphrase {
      random {
        length  = 40
        symbols = false
        digits  = true
      }
    }
  }
}
