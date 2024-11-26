data "blocks_data_source" "a_data_source" {
    a_map_of_resources {
        inner_string = "hi"
    }
}

resource "blocks_resource" "a_resource" {
    a_map_of_resources {
        inner_string = "hi"
    }
}