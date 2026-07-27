resource "simple_resource" "tagged" {
    for_each = toset(["alpha", "beta"])
    input_one = each.value
    input_two = false
}

output "alpha_result" {
    value = simple_resource.tagged["alpha"].result
}
