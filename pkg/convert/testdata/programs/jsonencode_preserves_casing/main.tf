# When converted, the object inside jsonencode should maintain the casing of the keys
output "data" {
  value = jsonencode({
    "foo": "bar",
    Content: "capitalized",
    "Quoted": "quoted",
    nested: [
      { "Key": "value" },
      { ANOTHER: "one" }
    ]
  })
}