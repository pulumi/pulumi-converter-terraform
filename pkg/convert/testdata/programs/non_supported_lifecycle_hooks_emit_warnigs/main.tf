resource "simple_resource" "a_resource" {
    input_one = "hello"
    input_two = true
    lifecycle {
        create_before_destroy = true
    }
}

resource "simple_resource" "b_resource" {
    input_one = "world"
    input_two = false
    lifecycle {
        replace_triggered_by = [simple_resource.a_resource]
    }
}