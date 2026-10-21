variable "prefix" {
  type = string
}

locals {
  value = "${var.prefix}-service"
}

output "value" {
  value = local.value
}
