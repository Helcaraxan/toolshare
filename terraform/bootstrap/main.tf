# This module contains the minimal Terraform required to automated infrastructure maintenance. It
# needs to be manually bootstrapped. After the initialization it can be maintained automatically
# through CI.

locals {
  labels = {
    bootstrap : "true"
    environment : "ci"
  }
}

resource "google_project_service" "this" {
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "sts.googleapis.com",
    "storage.googleapis.com",
  ])

  service                    = each.key
  disable_on_destroy         = false
  disable_dependent_services = false
}

resource "google_iam_workload_identity_pool" "this" {
  workload_identity_pool_id = "ci-automation"
  display_name              = "CI access-control"
  description               = "Used for external test & release automation providers"
}

resource "google_iam_workload_identity_pool_provider" "this" {
  # WARNING: Editing this resource will require a manual apply by a project owner. For security
  #          reasons the CI service-account does not have permissions to self-modify the provider
  #          which it uses to authenticate.
  workload_identity_pool_id          = google_iam_workload_identity_pool.this.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "GitHub"
  description                        = "WIP provider used for GitHub Action authentication"

  attribute_mapping = {
    "attribute.owner"      = "assertion.repository_owner"
    "attribute.owner_id"   = "assertion.repository_owner_id"
    "attribute.refs"       = "assertion.ref"
    "attribute.repository" = "assertion.repository"
    "google.subject"       = "assertion.sub"
  }

  attribute_condition = join(" && \n", [
    "attribute.owner == \"${var.github_owner}\"",
    "attribute.owner_id == \"${var.github_owner_id}\"",
    "attribute.repository == \"${var.github_slug}\"",
  ])

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }

  depends_on = [google_project_service.this]
}

resource "google_service_account" "this" {
  account_id   = "toolshare-ci"
  display_name = "Toolshare CI Automation"

  depends_on = [google_project_service.this]
}

resource "google_service_account_iam_member" "this" {
  service_account_id = google_service_account.this.id
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.this.name}/attribute.repository/${var.github_slug}"
  role               = "roles/iam.workloadIdentityUser"
}

resource "google_project_iam_member" "this" {
  for_each = toset([
    "roles/iam.securityAdmin",
    "roles/iam.workloadIdentityPoolViewer",
  ])

  project = var.google_project_id
  member  = google_service_account.this.member
  role    = each.value
}

resource "google_storage_bucket" "this" {
  name     = "toolshare-ci"
  location = var.google_region

  public_access_prevention    = "enforced"
  uniform_bucket_level_access = true
  requester_pays              = false

  versioning {
    enabled = true
  }

  hierarchical_namespace {
    enabled = false
  }

  enable_object_retention  = false
  default_event_based_hold = false
  soft_delete_policy {
    retention_duration_seconds = 604800
  }

  labels = local.labels

  depends_on = [google_project_service.this]
}

resource "google_storage_bucket_iam_member" "owner" {
  bucket = google_storage_bucket.this.id
  role   = "roles/storage.admin"
  member = google_service_account.this.member
}
