resource "simple_resource" "a_resource" {
    timeouts {
        create = "60m"
        delete = "2h"
        read = "1m"
        unrecognized = "5m"
    }

    input_one = "hello"
    input_two = true
}
