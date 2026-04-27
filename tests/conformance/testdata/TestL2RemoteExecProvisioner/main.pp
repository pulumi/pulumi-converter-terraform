config "conformanceKind" "string" {
}

config "sshPk" "string" {
}

config "sshPortInline" "number" {
}

config "sshPortScript" "number" {
}

config "sshPortScripts" "number" {
}

resource "exampleInline" "test:index/resource:Resource" {
  __logicalName = "example_inline"
  value         = "hello"
}
resource "exampleInlineProvisioner0" "command:remote:Command" {
  options {
    dependsOn = [exampleInline]
  }
  connection = {
    host       = "127.0.0.1"
    port       = sshPortInline
    privateKey = sshPk
    user       = conformanceKind
  }
  create = join("\n", ["echo ${exampleInline.computedValue}"])
}

resource "exampleScript" "test:index/resource:Resource" {
  __logicalName = "example_script"
  value         = "world"
}
exampleScriptProvisioner0Connection = {
  host       = "127.0.0.1"
  port       = sshPortScript
  privateKey = sshPk
  user       = conformanceKind
}
resource "exampleScriptProvisioner0Copy" "command:remote:CopyToRemote" {
  options {
    dependsOn = [exampleScript]
  }
  connection = exampleScriptProvisioner0Connection
  source     = fileAsset("./hello.sh")
  remotePath = "/tmp/exampleScriptProvisioner0"
}
resource "exampleScriptProvisioner0" "command:remote:Command" {
  options {
    dependsOn = [exampleScriptProvisioner0Copy]
  }
  connection = exampleScriptProvisioner0Connection
  create     = "/tmp/exampleScriptProvisioner0"
}

resource "exampleScripts" "test:index/resource:Resource" {
  __logicalName = "example_scripts"
  value         = "many"
}
exampleScriptsProvisioner0Connection = {
  host       = "127.0.0.1"
  port       = sshPortScripts
  privateKey = sshPk
  user       = conformanceKind
}
resource "exampleScriptsProvisioner0Copy0" "command:remote:CopyToRemote" {
  options {
    dependsOn = [exampleScripts]
  }
  connection = exampleScriptsProvisioner0Connection
  source     = fileAsset("./a.sh")
  remotePath = "/tmp/exampleScriptsProvisioner0/a.sh"
}
resource "exampleScriptsProvisioner0Copy1" "command:remote:CopyToRemote" {
  options {
    dependsOn = [exampleScriptsProvisioner0Copy0]
  }
  connection = exampleScriptsProvisioner0Connection
  source     = fileAsset("./b.sh")
  remotePath = "/tmp/exampleScriptsProvisioner0/b.sh"
}
resource "exampleScriptsProvisioner0" "command:remote:Command" {
  options {
    dependsOn = [exampleScriptsProvisioner0Copy1]
  }
  connection = exampleScriptsProvisioner0Connection
  create     = join("\n", ["/tmp/exampleScriptsProvisioner0/a.sh", "/tmp/exampleScriptsProvisioner0/b.sh"])
}
