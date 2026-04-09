config "rules" "list(object({port=number, protocol=string}))" {
  default = [{
    port     = 80
    protocol = "tcp"
    }, {
    port     = 443
    protocol = "tcp"
  }]
}

resource "example" "test:index/nestedResource:NestedResource" {
  rules = [for entry in rules : {
    port     = entry.port
    protocol = entry.protocol
  }]
  value = "test"
}

output "computed" {
  value = example.computedValue
}
