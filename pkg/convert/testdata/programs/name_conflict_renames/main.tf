resource "renames_resource" "a_thing" {
    a_number = 1
    a_resource {
        inner_string = "hello"
    }
}

data "renames_data_source" "a_thing" {
    a_number = 2
    a_resource {
        inner_string = "hello"
    }
}