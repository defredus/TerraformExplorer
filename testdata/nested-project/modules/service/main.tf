variable "name" {
  type = string
}

module "labels" {
  source = "../labels"
  prefix = var.name
}

resource "null_resource" "service" {
  triggers = {
    label = module.labels.value
  }
}

output "endpoint" {
  value = null_resource.service.id
}
