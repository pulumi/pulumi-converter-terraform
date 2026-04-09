config "name" "string" {
}

resource "this" "simple:index/resource:Resource" {
  inputOne = try(name, "default")
  inputTwo = true
}

output "missing" {
  value = try(this.result, 42)
}

output "present" {
  value = try(this.inputTwo, false)
}
