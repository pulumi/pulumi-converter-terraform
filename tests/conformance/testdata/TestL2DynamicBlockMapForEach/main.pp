config "ruleMap" "map(object({port=number}))" {
  default = {
    tcp = {
      port = 80
    }
    udp = {
      port = 53
    }
  }
}

resource "example" "test:index/nestedResource:NestedResource" {
  rules = [for key, entry in ruleMap : {
    port     = entry.port
    protocol = key
  }]
  value = "test"
}

output "computed" {
  value = example.computedValue
}
