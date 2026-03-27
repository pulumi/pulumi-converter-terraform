resource "example" "test:index/resource:Resource" {
  options {
    ignoreChanges = [value]
  }
  value = "hello"
}

resource "computedOnly" "test:index/resource:Resource" {
  __logicalName = "computed_only"
  value         = "world"
}

resource "bridgeComputed" "test:index/taggedResource:TaggedResource" {
  __logicalName = "bridge_computed"
  value         = "tagged"
}

output "result" {
  value = example.computedValue
}
