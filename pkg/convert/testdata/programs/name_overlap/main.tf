resource "simple_resource" "for" {
    input_one = "hello"
    input_two = true
}

resource "simple_resource" "dependsOnFor" {
    depends_on = [simple_resource.for]

    input_one = simple_resource.for.result
    input_two = false
}
