variable "listvar" {
  default = ["2025"]
}

variable "dynvar" {
  type = list(object({
    inner_value = string
    inner_list = list(object({
      nested_value = string
    }))
  }))
  default = []
}

resource "blocks_resource" "a_resource" {
    count = length(listvar) > 0 ? 1 : 0
    dynamic "a_list_of_resources" {
        for_each = var.dynvar

        content {
            inner_string = a_list_of_resources.value.inner_value != null ? "TrySuccess" : "TryFail"
            dynamic "inner_dynamic_resources" {
              for_each = a_list_of_resources.value.inner_list
              content {
                # Utilize the inner resource in the inner dynamic block, this
                # worked even before pulumi/pulumi#18718 was fixed.
                nested_string = inner_dynamic_resources.value.nested_value
              }
            }
        }
    }
}

resource "blocks_resource" "b_resource" {
    count = length(listvar) > 0 ? 1 : 0
    dynamic "a_list_of_resources" {
        for_each = var.dynvar

        content {
            inner_string = a_list_of_resources.value.inner_value != null ? "TrySuccess" : "TryFail"
            dynamic "inner_dynamic_resources" {
              for_each = a_list_of_resources.value.inner_list
              content {
                # This was fixed by pulumi/pulumi#18718.  Before the generated
                # PCL would shadow a_list_of_resources with the same iterator
                # name (entry). 
                nested_string = a_list_of_resources.value.inner_value
              }
            }
        }
    }
}
