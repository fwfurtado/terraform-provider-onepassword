# ephemeral "onepassword_vault" "this" {
#   vault = "homelab"
# }

# ephemeral "onepassword_secret" "this" {
#   reference = "op://homelab/Grafana/password"
# }

# ephemeral "onepassword_item_overview" "this" {
#   vault_id = ephemeral.onepassword_vault.this.id
#   name = "Grafana"
# }

ephemeral "onepassword_item" "this" {
    vault = "homelab"
    name = "Grafana"
}