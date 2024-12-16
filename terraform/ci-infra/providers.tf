terraform {
  required_version = "~>1.10.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~>6.11.2"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~>2.7.0"
    }
    null = {
      source  = "hashicorp/null"
      version = "~>3.2.3"
    }
  }

  backend "gcs" {
    bucket = "toolshare-ci"
    prefix = "tf-state/ci"
  }
}

provider "google" {
  project = var.google_project
  region  = var.google_region
}
