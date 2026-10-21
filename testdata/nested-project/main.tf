variable "name" {
  default = "explorer"
}

module "service" {
  source = "./modules/service"
  name   = var.name
}

module "remote" {
  source = "hashicorp/consul/aws"
}

output "endpoint" {
  value = module.service.endpoint
}
