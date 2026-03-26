config "env" "string" {
}

output "staticMap" {
  value = {
    name = "alice"
    age  = "30"
  }
}

output "dynamicMap" {
  value = {
    environment = env
    region      = "us-east-1"
  }
}
