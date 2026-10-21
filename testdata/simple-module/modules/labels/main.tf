variable "project" {
  type = string
}

locals {
  application = "${var.project}-explorer"
}

output "application" {
  value = local.application
}
