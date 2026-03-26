config "name" "string" {
}

output "doc" {
  value = <<-EOT
Hello, ${name}!
This is a heredoc.
EOT

}
