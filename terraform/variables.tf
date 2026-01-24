variable "service_account_token" {
  type        = string
  description = "1Password Service Account token"
  ephemeral   = true
}

variable "desktop_account_name" {
  type        = string
  description = "1Password Desktop account name"
  ephemeral   = true
}

variable "vault" {
  type        = string
  description = "1Password vault name"
}