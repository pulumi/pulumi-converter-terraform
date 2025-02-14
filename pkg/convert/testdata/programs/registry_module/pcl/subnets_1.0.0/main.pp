addrsByIdx = invoke("std:index:cidrsubnets", {
  input   = baseCidrBlock
  newbits = networks[*].newBits
}).result
addrsByName = { for i, n in networks : n.name => addrsByIdx[i] if n.name != null }
networkObjs = [for i, n in networks : {
  name      = n.name
  newBits   = n.newBits
  cidrBlock = n.name != null ? addrsByIdx[i] : notImplemented("tostring(null)")
}]
