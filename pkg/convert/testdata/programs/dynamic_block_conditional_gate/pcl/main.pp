config "includeInner" "bool" {
  default = true
}

resource "this" "blocks:index/index:resource" {
  aListOfResources = includeInner ? [{
    innerString = "hello"
  }] : []
}
