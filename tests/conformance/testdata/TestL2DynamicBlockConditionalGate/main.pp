config "includeRule" "bool" {
  default = true
}

resource "example" "test:index/nestedResource:NestedResource" {
  rules = includeRule ? [{
    port     = 80
    protocol = "tcp"
  }] : []
  value = "test"
}

output "computed" {
  value = example.computedValue
}
