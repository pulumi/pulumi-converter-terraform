variable "include_inner" {
    type    = bool
    default = true
}

resource "blocks_resource" "this" {
    dynamic "a_list_of_resources" {
        for_each = var.include_inner ? ["hello"] : []
        content {
            inner_string = a_list_of_resources.value
        }
    }
}
