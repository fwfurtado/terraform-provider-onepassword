resource "onepassword_software_license" "example" {
  vault   = "Engineering"
  version = 1
  tags    = ["terraform", "example"]

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
