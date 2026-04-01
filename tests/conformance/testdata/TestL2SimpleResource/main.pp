resource "aResource" "simple:index/resource:Resource" {
  __logicalName = "a_resource"
  inputOne      = "hello"
  inputTwo      = true
}

output "someOutput" {
  value = aResource.result
}
