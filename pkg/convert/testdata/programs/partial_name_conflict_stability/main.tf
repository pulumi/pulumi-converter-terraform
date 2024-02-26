resource "simple_resource" "a_thing" {
    input_one = "Hello ${simple_resource.test.result}"
    input_two = complex_resource.test.result
}
