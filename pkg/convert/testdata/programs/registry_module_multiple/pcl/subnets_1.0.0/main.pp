addrs_by_idx  = notImplemented("cidrsubnets(var.base_cidr_block,var.networks[*].new_bits...)")
addrs_by_name = { for i, n in networks : n.name => addrs_by_idx[i] if n.name != null }
network_objs = [for i, n in networks : {
  name       = n.name
  new_bits   = n.new_bits
  cidr_block = n.name != null ? addrs_by_idx[i] : notImplemented("tostring(null)")
}]
