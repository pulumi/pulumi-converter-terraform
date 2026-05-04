# Test for `file` provisioner conversion. Covers source/content forms and
# verifies that a connection-level `timeout` is propagated as a `customTimeouts`
# option on the generated CopyToRemote resource — both when the connection
# lives on the parent resource and when it is overridden on the provisioner
# itself.
resource "fileResource" "simple:index:resource" {
  __logicalName = "file_resource"
  inputOne      = "hello"
  inputTwo      = true

}
resource "fileResourceProvisioner0" "command:remote:CopyToRemote" {
  options {
    dependsOn = [fileResource]
    customTimeouts = {
      create = "30s"
      update = "30s"
    }
  }
  connection = {
    host       = "primary.example.com"
    privateKey = "resource-key"
    user       = "deploy"
  }
  source     = stringAsset("from inline content\n")
  remotePath = "/tmp/file-provisioner-content.txt"
}
resource "fileResourceProvisioner1" "command:remote:CopyToRemote" {
  options {
    dependsOn = [fileResourceProvisioner0]
    customTimeouts = {
      create = "2m"
      update = "2m"
    }
  }
  connection = {
    host       = "secondary.example.com"
    privateKey = "override-key"
    user       = "deploy"
  }
  source     = try(fileAsset("./payload.txt"), fileArchive("./payload.txt"))
  remotePath = "/tmp/file-provisioner-source.txt"
}
