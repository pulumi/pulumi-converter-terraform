resource "aResource" "complex:index/index:resource" {
  __logicalName = "a_resource"
  aMapOfBool = {
    CAPS       = true
    camelCase  = false
    snake_case = false
    PascalCase = true
  }
}

output "someMapOutput" {
  value = {
    CAPS       = aResource.result
    camelCase  = 1
    snakeCase  = 2
    pascalCase = 3
  }
}
