resource "onepassword_document" "example" {
  vault = "Engineering"
  name  = "Example Document"

  document {
    filename = "example.txt"
    content  = file("${path.module}/example.txt")
  }

  sections = {
    "metadata" = {
      label = "Metadata"
      fields = [
        {
          label = "owner"
          type  = "text"
          value = "Example Owner"
        }
      ]
    }
  }
}
