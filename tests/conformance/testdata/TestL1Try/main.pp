config "config" "object({name=string, nested=object({value=string})})" {
}

output "name" {
  value = config.name
}

output "nestedOrFallback" {
  value = try(config.nested.value, "fallback")
}
