resource "simple_resource" "a_resource" {
    input_one = "hello"
    input_two = true
}

resource "simple_resource" "b_resource" {
    depends_on = [simple_resource.a_resource]

    input_one = simple_resource.a_resource.result
    input_two = false
}