variable "google_project" {
  type        = string
  description = "GCP project where to provision all resources."
  sensitive   = true
}

variable "google_region" {
  type        = string
  description = "GCP region where to provision resources."
  default     = "us-east1"
}

variable "github_slug" {
  type        = string
  description = "Full GitHub slug of the repository containing the Toolshare source code."
}

variable "github_owner" {
  type        = string
  description = "Owner name of the repository containing the Toolshare source code."
}

variable "github_owner_id" {
  type        = string
  description = "ID of the owner of the repository containing the Toolshare source code."
}
