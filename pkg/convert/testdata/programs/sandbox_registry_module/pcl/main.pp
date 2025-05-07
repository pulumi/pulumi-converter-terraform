package "subnets" {
  baseProviderName    = "terraform-module"
  baseProviderVersion = "0.1.3"
  parameterization {
    name    = "subnets"
    version = "1.0.0"
    value   = "eyJtb2R1bGUiOiJyZWdpc3RyeS50ZXJyYWZvcm0uaW8vaGFzaGljb3JwL3N1Ym5ldHMvY2lkciIsInBhY2thZ2VOYW1lIjoic3VibmV0cyIsInZlcnNpb24iOiIxLjAuMCJ9"
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

output "blocks" {
  value = subnetsCidr.network_cidr_blocks
}
