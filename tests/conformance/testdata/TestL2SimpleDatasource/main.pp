aDataSource = invoke("simple:index/getDataSource:getDataSource", {
  inputOne = "hello"
  inputTwo = true
})

output "someOutput" {
  value = aDataSource.result
}
