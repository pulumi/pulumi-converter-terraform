module "some_module" {
    source = "./mod"
}

output "module_path_output" {
    value = module.some_module.output
}

output "root_output" {
    value = path.root
}

output "cwd_output" {
    value = path.cwd
}

output "workspace_output" {
    value = terraform.workspace
}
