resource "aThing" "simple:index:resource" {
  __logicalName = "a_thing"
  inputOne      = "Hello ${test.result}"
  inputTwo      = testComplexResource.result
}
