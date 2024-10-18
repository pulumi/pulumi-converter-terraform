
// Check we can use terraform builtin functions here and that they are evaluated
static_local = invoke("std:index:title", {
  input = "static"
}).result

resource "aDefaultResource" "configured:index:resource" {
  __logicalName = "a_default_resource"
  inputOne      = static_local
}
