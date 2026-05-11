config "config" "object({name=string, nested=object({value=string})})" {
}

output "name" {
  value = config.name
}

output "hasNested" {
  value = can(config.nested.value)
}
