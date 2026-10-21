variable "project" {
  description = "Project name"
  type        = string
  default     = "explorer"
}

locals {
  full_name = "${var.project}-assets"
  enabled   = true
}

data "aws_caller_identity" "current" {}

resource "aws_s3_bucket" "assets" {
  bucket = local.full_name
}

output "bucket_id" {
  value = aws_s3_bucket.assets.id
}
