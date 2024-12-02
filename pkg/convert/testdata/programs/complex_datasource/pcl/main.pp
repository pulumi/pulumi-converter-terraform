aDataSource = invoke("complex:index/index:dataSource", {
  aBool       = true
  aNumber     = 2.3
  aString     = "hello world"
  aListOfInts = [1, 2, 3]
  aMapOfBool = {
    a = true
    b = false
  }
  innerListObject = {
    innerString = "hello again"
  }
  innerMapObject = {
    innerString = "hello thrice"
  }
})

output "someOutput" {
  value = aDataSource.result
}
