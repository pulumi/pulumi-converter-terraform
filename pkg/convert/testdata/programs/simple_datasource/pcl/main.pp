aDataSource = invoke("simple:index:dataSource", {
  inputOne = "hello"
  inputTwo = true
})

output "someOutput" {
  value = aDataSource.result
}
