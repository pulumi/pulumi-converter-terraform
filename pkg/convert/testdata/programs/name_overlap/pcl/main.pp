resource "forResource_" "simple:index:resource" {
  __logicalName = "for"
  inputOne      = "hello"
  inputTwo      = true
}

resource "dependsOnFor" "simple:index:resource" {
  options {
    dependsOn = [forResource_]
  }
  inputOne = forResource_.result
  inputTwo = false
}
