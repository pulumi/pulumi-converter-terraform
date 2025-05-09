// @pulumi-terraform-module subnets
module "subnets_cidr" {
  source = "hashicorp/subnets/cidr"
  version = "1.0.0"

  base_cidr_block = "10.0.0.0/8"
  networks = [
    {
      name     = "foo"
      new_bits = 8
    },
    {
      name     = "bar"
      new_bits = 8
    },
  ]
}

//@pulumi-terraform-module subnets
module "another_subnets_cidr" {
  source = "hashicorp/subnets/cidr"
  version = "1.0.0"
  base_cidr_block = "10.0.0.0/16"
}

module "without_annotation" {
  source = "hashicorp/subnets/cidr"
  version = "1.0.0"
  base_cidr_block = "10.0.0.0/32"
  count = 10
}

output "blocks" {
    value = module.subnets_cidr.network_cidr_blocks
}