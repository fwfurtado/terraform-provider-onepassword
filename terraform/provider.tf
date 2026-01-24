terraform {
  required_providers {
    onepassword = {
      source = "registry.terraform.io/fwfurtado/onepassword"
    }
  }
}

provider "onepassword" {
  service_account {
    token = var.service_account_token
  }

  default_tags {
    tags = {
      # quaquer par de chave-valor pode ser adicionado aqui
      Environment = "Development"
      Project     = "OnePassword"
      Owner       = "John Doe"
      CreatedBy   = "John Doe"
      CreatedAt   = "2021-01-01"
      UpdatedAt   = "2021-01-01"
      UpdatedBy   = "John Doe"
      UpdatedBy   = "John Doe"
    }
  }
}