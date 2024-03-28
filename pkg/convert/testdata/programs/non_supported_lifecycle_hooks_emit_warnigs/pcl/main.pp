resource "aResource" "simple:index:resource" {
  __logicalName = "a_resource"
  inputOne      = "hello"
  inputTwo      = true
}

resource "bResource" "simple:index:resource" {
  __logicalName = "b_resource"
  inputOne      = "world"
  inputTwo      = false
}
