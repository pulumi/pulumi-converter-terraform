config "tags" "map(string)" {
}

config "ports" "list(number)" {
}

config "config" "object({enabled=bool, name=string})" {
}

output "tagEnv" {
  value = tags["env"]
}

output "firstPort" {
  value = ports[0]
}

output "configName" {
  value = config.name
}
