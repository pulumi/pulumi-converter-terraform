component "someModule" "./mod" {
}

output "modulePathOutput" {
  value = someModule.output
}

output "rootOutput" {
  value = rootDirectory()
}

output "cwdOutput" {
  value = cwd()
}

output "workspaceOutput" {
  value = notImplemented("terraform.workspace")
}
