terraform {
  required_version = "~>1.10.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~>6.11.2"
    }
  }

  backend "gcs" {
    bucket = "toolshare-ci"
    prefix = "tf-state/bootstrap"
  }
}

provider "google" {
  project = var.google_project_id
  region  = var.google_region
}
