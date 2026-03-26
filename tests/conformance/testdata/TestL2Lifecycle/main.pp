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

output "result" {
  value = example.computedValue
}
