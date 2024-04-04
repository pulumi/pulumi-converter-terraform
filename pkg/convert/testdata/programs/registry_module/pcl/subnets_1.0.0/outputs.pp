output "networkCidrBlocks" {
  value = notImplemented("tomap(local.addrs_by_name)")
}

output "networks" {
  value = networkObjs
}

output "baseCidrBlock" {
  value = baseCidrBlock
}
