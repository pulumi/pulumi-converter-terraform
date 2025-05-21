// @pulumi-terraform-module example
module "exampleModule" {
    source = "./example"
    name   = "John"
}

output "name" {
    value = module.exampleModule.name
}