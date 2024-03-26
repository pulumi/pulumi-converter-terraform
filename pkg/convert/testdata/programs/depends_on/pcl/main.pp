resource "aResource" "simple:index:resource" {
  __logicalName = "a_resource"
  inputOne      = "hello"
  inputTwo      = true
}

resource "bResource" "simple:index:resource" {
  __logicalName = "b_resource"
  options {
    dependsOn = [aResource]
  }
  inputOne = aResource.result
  inputTwo = false
}
