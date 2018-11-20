terraform {
  # Version of Terraform to include in the bundle. An exact version number
  # is required.
  version = "0.11.10"
}

# Define which provider plugins are to be included
providers {
  # Include the newest 1.0 version of the "google" provider.
  google = [">= 1.9"]
}
