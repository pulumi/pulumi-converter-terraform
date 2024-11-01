output "files" {
  value = files
}

output "filesOnDisk" {
  value = { for p, f in files : p => f if f.source_path != null }
}

output "filesInMemory" {
  value = { for p, f in files : p => f if f.content != null }
}
