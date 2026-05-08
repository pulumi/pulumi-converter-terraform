resource "aResource" "simple:index:resource" {
  __logicalName = "a_resource"
  options {
    customTimeouts = { // dropped unsupported timeouts attribute(s): read, unrecognized

      create = "60m"
      delete = "2h"
    }
  }
  inputOne = "hello"
  inputTwo = true
}
