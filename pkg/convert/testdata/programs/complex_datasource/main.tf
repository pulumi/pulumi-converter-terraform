data "complex_data_source" "a_data_source" {
    a_bool = true
    a_number = 2.3
    a_string = "hello world"
    a_list_of_int = [1, 2, 3]
    a_map_of_bool = {
        a: true
        b: false
    }
    inner_list_object = [{
        inner_string = "hello again"
    }]
    inner_map_object = {
        inner_string = "hello thrice"
    }
}

output "some_output" {
    value = data.complex_data_source.a_data_source.result
}