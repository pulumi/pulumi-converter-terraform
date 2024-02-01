output "networkCidrBlocks" {
  value = notImplemented("tomap(local.addrs_by_name)")
}

output "networks" {
  value = invoke("std:index:tolist", {
    input = networkObjs
  }).result
}

output "baseCidrBlock" {
  value = baseCidrBlock
}
