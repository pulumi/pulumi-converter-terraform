example = invoke("test:index/getData:getData", {
  input = "hello"
})

output "result" {
  value = example.result
}
