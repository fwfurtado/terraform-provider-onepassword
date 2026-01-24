terraform {
  required_providers {
    onepassword = {
      source = "registry.terraform.io/fwfurtado/onepassword"
    }
  }
}

provider "onepassword" {
  service_account = {
    token = var.service_account_token
  }

}