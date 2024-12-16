locals {
  labels = {
    bootstrap : "false"
    environment : "ci"
  }
  repo_root = "${path.module}/../.."
}

resource "google_project_service" "this" {
  for_each = toset([
    "storage.googleapis.com",
  ])

  service = each.value
}

resource "google_storage_bucket" "test-artefacts" {
  name     = "toolshare-test-storage"
  location = var.google_region

  public_access_prevention    = "enforced"
  uniform_bucket_level_access = true
  requester_pays              = false

  versioning {
    enabled = false
  }

  hierarchical_namespace {
    enabled = false
  }

  enable_object_retention  = false
  default_event_based_hold = false
  soft_delete_policy {
    retention_duration_seconds = 0
  }

  labels = local.labels

  depends_on = [google_project_service.this]
}

data "archive_file" "testdata" {
  type        = "zip"
  source_dir  = "${local.repo_root}/internal/driver/testdata"
  output_path = "${local.repo_root}/internal/driver/.testdata.zip"
}

resource "null_resource" "e2e-test-artefacts" {
  provisioner "local-exec" {
    command = join(" ", [
      "gsutil -m rsync -rd",
      "${local.repo_root}/internal/driver/testdata",
      "gs://${google_storage_bucket.test-artefacts.name}/",
    ])
  }
  triggers = {
    testdata_hash : data.archive_file.testdata.output_sha256
  }
}
