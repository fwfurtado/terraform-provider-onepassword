resource "onepassword_database" "example" {
  vault   = "Engineering"
  version = 1
  tags    = ["terraform", "example"]

  name = "Example Database"

  type               = "postgresql"
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
