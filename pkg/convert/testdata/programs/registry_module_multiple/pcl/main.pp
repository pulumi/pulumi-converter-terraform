// This test checks that if we have the same module used twice, but it doesn't have _exactly_ the same version string
// (and so a different key) that it still works correctly.
component "cidrsOne" "./subnets_1.0.0" {
  baseCidrBlock = "10.0.0.0/8"
  networks = [{
    name    = "foo"
    newBits = 8
  }]
}

component "cidrsTwo" "./subnets_1.0.0" {
  baseCidrBlock = "10.0.0.0/8"
  networks = [{
    name    = "bar"
    newBits = 8
  }]
}

output "blocksOne" {
  value = cidrsOne.networkCidrBlocks
}

output "blocksTwo" {
  value = cidrsTwo.networkCidrBlocks
}
