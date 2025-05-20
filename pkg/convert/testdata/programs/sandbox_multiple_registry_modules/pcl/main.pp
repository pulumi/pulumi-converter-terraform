package "subnets" {
  baseProviderName    = "terraform-module"
  baseProviderVersion = "0.1.4"
  parameterization {
    name    = "subnets"
    version = "1.0.0"
    // encoded parameterization values:
    // module: registry.terraform.io/hashicorp/subnets/cidr
    // version: 1.0.0
    // packageName: subnets
    value = "eyJtb2R1bGUiOiJyZWdpc3RyeS50ZXJyYWZvcm0uaW8vaGFzaGljb3JwL3N1Ym5ldHMvY2lkciIsInBhY2thZ2VOYW1lIjoic3VibmV0cyIsInZlcnNpb24iOiIxLjAuMCJ9"
  }
}

resource "subnetsCidr" "subnets:index:Module" {
  base_cidr_block = "10.0.0.0/8"
  networks = [{
    name     = "foo"
    new_bits = 8
    }, {
    name     = "bar"
    new_bits = 8
  }]
}
resource "anotherSubnetsCidr" "subnets:index:Module" {
  base_cidr_block = "10.0.0.0/16"
}
resource "withoutAnnotation" "subnets:index:Module" {
  options {
    range = 10
  }
  base_cidr_block = "10.0.0.0/32"
}

output "blocks" {
  value = subnetsCidr.network_cidr_blocks
}
