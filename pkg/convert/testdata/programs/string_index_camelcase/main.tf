locals {
    items = {
        foo = {
            aws_region    = "us-east-1"
            instance_type = "t2.micro"
        }
        bar = {
            aws_region    = "us-west-2"
            instance_type = "t2.small"
        }
    }
}

# Test 1: Direct string index on locals with bracket notation
output "local_string_index" {
    value = local.items["foo"]["aws_region"]
}

# Test 2: Using each.value with bracket notation in for_each
resource "simple_resource" "test_for_each" {
    for_each = local.items
    input_one = each.value["aws_region"]
    input_two = 0
}

# Test 3: Using range.value with bracket notation in for expression
output "for_expr_string_index" {
    value = { for key, value in local.items : key => value["instance_type"] }
}
