variable "google_project" {
  type        = string
  description = "GCP project where to provision all resources."
  default     = "carbide-atlas-327122"
}

variable "google_region" {
  type        = string
  description = "GCP region where to provision resources."
  default     = "us-east1"
}
