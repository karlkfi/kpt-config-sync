terraform {
  # Version of Terraform to include in the bundle. An exact version number
  # is required.
  version = "0.11.10"
}

# Define which provider plugins are to be included
providers {
  # See https://releases.hashicorp.com/terraform-provider-google
  # This version number should be the same as in const providerConfig
  # in pkg/bespin-controllers/terraform/executor.go
  google = ["1.19.1"]
}
