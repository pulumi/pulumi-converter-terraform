resource "forResourcePcl" "simple:index:resource" {
  __logicalName = "for_resource_pcl"
  inputOne      = "hello"
  inputTwo      = true
}

resource "dependsOnFor" "simple:index:resource" {
  options {
    dependsOn = [forResourcePcl]
  }
  inputOne = forResourcePcl.result
  inputTwo = false
}
