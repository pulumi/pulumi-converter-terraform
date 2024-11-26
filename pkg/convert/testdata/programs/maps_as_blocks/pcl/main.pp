aDataSource = invoke("blocks:index/index:dataSource", {
  aMapOfResources = {
    innerString = "hi"
  }
})

resource "aResource" "blocks:index/index:resource" {
  __logicalName = "a_resource"
  aMapOfResources = {
    innerString = "hi"
  }
}
