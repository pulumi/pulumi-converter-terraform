variable "a_thing" {

}

locals {
    aThing = true
}

resource "simple_resource" "a_thing" {
    input_one = "Hello ${var.a_thing}"
    input_two = local.aThing
}

data "simple_data_source" "a_thing" {
    input_one = "Hello ${simple_resource.a_thing.result}"
    input_two = local.aThing
}

resource "simple_another_resource" "a_thing" {
    input_one = "Hello ${simple_resource.a_thing.result}"
}

output "a_thing" {
    value = data.simple_data_source.a_thing.result
}
