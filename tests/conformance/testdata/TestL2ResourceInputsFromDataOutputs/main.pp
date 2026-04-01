example = invoke("simple:index/getDataSource:getDataSource", {
  inputOne = "hello"
  inputTwo = true
})

resource "exampleResource" "simple:index/resource:Resource" {
  __logicalName = "example"
  inputOne      = example.inputOne
  inputTwo      = example.inputTwo
}

output "example" {
  value = exampleResource.result
}
