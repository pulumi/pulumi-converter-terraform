aDataSource = invoke("blocks:index/index:dataSource", {
  aListOfResources = [{
    innerString = "hi"
    }, {
    innerString = "bye"
  }]
})

resource "aResource" "blocks:index/index:resource" {
  __logicalName = "a_resource"
  aListOfResources = [{
    innerString = "hi"
    }, {
    innerString = "bye"
  }]
}
