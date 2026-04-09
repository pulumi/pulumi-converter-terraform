resource "aResource" "blocks:index/index:resource" {
  __logicalName = "a_resource"
  aListOfResources = [for entry in ["hi", "bye"] : {
    innerString = entry
  }]
}

resource "bResource" "blocks:index/index:resource" {
  __logicalName = "b_resource"
  aListOfResources = [for entry in ["hi", "bye"] : {
    innerString = entry
  }]
}

resource "cResource" "blocks:index/index:resource" {
  __logicalName = "c_resource"
  aListOfResources = [for entry in ["hi", "bye"] : {
    innerString = entry
  }]
}
