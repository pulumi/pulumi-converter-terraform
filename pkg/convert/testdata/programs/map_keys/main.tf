resource "complex_resource" "a_resource" {
    a_map_of_bool = {
        CAPS: true
        camelCase: false
        snake_case: false
        PascalCase: true
    }
}

output "some_map_output" {
    value = {
        CAPS: complex_resource.a_resource.result
        camelCase: 1
        snake_case: 2
        PascalCase: 3
    }
}