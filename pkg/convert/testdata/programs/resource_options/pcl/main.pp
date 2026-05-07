resource "aResource" "simple:index:resource" {
  __logicalName = "a_resource"
  options {
    customTimeouts = {
      create = "60m"
      delete = "2h"
    }
  }
  inputOne = "hello"
  inputTwo = true
}
