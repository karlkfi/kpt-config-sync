terraform {
  # Version of Terraform to include in the bundle. An exact version number
  # is required.
  version = "0.11.10"
}

# Define which provider plugins are to be included
providers {
  # See https://releases.hashicorp.com/terraform-provider-google
  google = ["1.19.1"]
}
