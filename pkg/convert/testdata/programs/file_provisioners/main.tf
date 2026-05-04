# Test for `file` provisioner conversion. Covers source/content forms and
# verifies that a connection-level `timeout` is propagated as a `customTimeouts`
# option on the generated CopyToRemote resource — both when the connection
# lives on the parent resource and when it is overridden on the provisioner
# itself.

resource "simple_resource" "file_resource" {
  input_one = "hello"
  input_two = true

  connection {
    host        = "primary.example.com"
    user        = "deploy"
    private_key = "resource-key"
    timeout     = "30s"
  }

  # Inherits the resource-level connection (and its 30s timeout).
  provisioner "file" {
    content     = "from inline content\n"
    destination = "/tmp/file-provisioner-content.txt"
  }

  # Overrides the resource-level connection with one that has its own,
  # longer timeout. The override should also override the timeout.
  provisioner "file" {
    connection {
      host        = "secondary.example.com"
      user        = "deploy"
      private_key = "override-key"
      timeout     = "2m"
    }
    source      = "./payload.txt"
    destination = "/tmp/file-provisioner-source.txt"
  }
}
