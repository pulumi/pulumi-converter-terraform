// This test checks that if we have the same module used twice, but it doesn't have _exactly_ the same version string
// (and so a different key) that it still works correctly.

module "cidrs_one" {
  source = "hashicorp/subnets/cidr"
  version = ">= 1.0.0, < 1.0.1"

  base_cidr_block = "10.0.0.0/8"
  networks = [
    {
      name     = "foo"
      new_bits = 8
    },
  ]
}

module "cidrs_two" {
  source = "hashicorp/subnets/cidr"
  version = "1.0.0"

  base_cidr_block = "10.0.0.0/8"
  networks = [
    {
      name     = "bar"
      new_bits = 8
    },
  ]
}

output "blocks_one" {
    value = module.cidrs_one.network_cidr_blocks
}

output "blocks_two" {
    value = module.cidrs_two.network_cidr_blocks
}