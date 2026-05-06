// Copyright 2026, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package conformance

import (
	"testing"

	"github.com/pulumi/pulumi-converter-terraform/pkg/testing/conformance"
	"github.com/pulumi/pulumi-converter-terraform/pkg/testing/sshcontainer"
	"github.com/pulumi/pulumi-converter-terraform/tests/conformance/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestL2FileProvisionerSourceFile verifies that a `file` provisioner whose
// `source` is a literal path resolving to an on-disk file converts to a
// CopyToRemote with `source = fileAsset(...)` and copies the file to the
// remote.
func TestL2FileProvisionerSourceFile(t *testing.T) {
	t.Parallel()
	installPulumiCommandPlugin(t)

	ssh := sshcontainer.Start(t)

	_, err := ssh.Exec("mkdir -p /tmp/conformance")
	require.NoError(t, err, "pre-creating remote dir")

	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Config: remoteExecConfig(ssh),
		Input: map[string]string{
			"main.tf": remoteExecConnectionVars + `
resource "test_resource" "example" {
  value = "hello"

  provisioner "file" {
    connection {
      host        = var.ssh_host
      port        = var.ssh_port
      user        = var.ssh_user
      private_key = var.ssh_private_key
    }
    source      = "./hello.txt"
    destination = "/tmp/conformance/source-file-${var.conformance_kind}.txt"
  }
}

output "value" {
  value = test_resource.example.value
}
`,
			"hello.txt": "hello from file provisioner\n",
		},
	})

	for _, kind := range []string{"tf", "pulumi"} {
		out, err := ssh.Exec("cat /tmp/conformance/source-file-" + kind + ".txt")
		require.NoError(t, err, "reading %s remote file", kind)
		assert.Equal(t, "hello from file provisioner\n", out, "%s file contents", kind)
	}
}

// TestL2FileProvisionerSourceDirNoTrailingSlash verifies that a `file`
// provisioner whose `source` is a literal directory path *without* a trailing
// slash converts to `source = fileArchive(...)` and that the directory itself
// is copied under the destination on the remote (matching Terraform semantics).
func TestL2FileProvisionerSourceDirNoTrailingSlash(t *testing.T) {
	t.Parallel()
	installPulumiCommandPlugin(t)

	ssh := sshcontainer.Start(t)

	// Pre-create the destination directories on the remote so the test can focus
	// purely on the `file` provisioner's behavior.
	for _, kind := range []string{"tf", "pulumi"} {
		_, err := ssh.Exec("mkdir -p /tmp/conformance/dir-no-slash-" + kind)
		require.NoError(t, err, "pre-creating %s remote dir", kind)
	}

	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Config: remoteExecConfig(ssh),
		Input: map[string]string{
			"main.tf": remoteExecConnectionVars + `
resource "test_resource" "example" {
  value = "hello"

  provisioner "file" {
    connection {
      host        = var.ssh_host
      port        = var.ssh_port
      user        = var.ssh_user
      private_key = var.ssh_private_key
    }
    source      = "./payload"
    destination = "/tmp/conformance/dir-no-slash-${var.conformance_kind}"
  }
}

output "value" {
  value = test_resource.example.value
}
`,
			"payload/a.txt": "alpha\n",
			"payload/b.txt": "bravo\n",
		},
	})

	// Without a trailing slash on source, Terraform copies the directory itself
	// under the destination, so the files end up at <destination>/payload/<file>.
	for _, kind := range []string{"tf", "pulumi"} {
		out, err := ssh.Exec("cat /tmp/conformance/dir-no-slash-" + kind + "/payload/a.txt")
		require.NoError(t, err, "%s a.txt", kind)
		assert.Equal(t, "alpha\n", out)

		out, err = ssh.Exec("cat /tmp/conformance/dir-no-slash-" + kind + "/payload/b.txt")
		require.NoError(t, err, "%s b.txt", kind)
		assert.Equal(t, "bravo\n", out)
	}
}

// TestL2FileProvisionerSourceDirTrailingSlash verifies that a `file`
// provisioner whose `source` is a literal directory path *with* a trailing
// slash converts to `source = fileArchive(...)` and that just the *contents*
// land at the destination (matching Terraform semantics).
func TestL2FileProvisionerSourceDirTrailingSlash(t *testing.T) {
	t.Parallel()
	installPulumiCommandPlugin(t)

	ssh := sshcontainer.Start(t)

	for _, kind := range []string{"tf", "pulumi"} {
		_, err := ssh.Exec("mkdir -p /tmp/conformance/dir-slash-" + kind)
		require.NoError(t, err, "pre-creating %s remote dir", kind)
	}

	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Config: remoteExecConfig(ssh),
		Input: map[string]string{
			"main.tf": remoteExecConnectionVars + `
resource "test_resource" "example" {
  value = "hello"

  provisioner "file" {
    connection {
      host        = var.ssh_host
      port        = var.ssh_port
      user        = var.ssh_user
      private_key = var.ssh_private_key
    }
    source      = "./payload/"
    destination = "/tmp/conformance/dir-slash-${var.conformance_kind}"
  }
}

output "value" {
  value = test_resource.example.value
}
`,
			"payload/a.txt": "alpha\n",
			"payload/b.txt": "bravo\n",
		},
	})

	// With a trailing slash on source, Terraform copies the *contents* of the
	// directory to the destination, so the files end up directly at
	// <destination>/<file>.
	for _, kind := range []string{"tf", "pulumi"} {
		out, err := ssh.Exec("cat /tmp/conformance/dir-slash-" + kind + "/a.txt")
		require.NoError(t, err, "%s a.txt", kind)
		assert.Equal(t, "alpha\n", out)

		out, err = ssh.Exec("cat /tmp/conformance/dir-slash-" + kind + "/b.txt")
		require.NoError(t, err, "%s b.txt", kind)
		assert.Equal(t, "bravo\n", out)
	}
}

// TestL2FileProvisionerContent verifies that a `file` provisioner using the
// `content` form converts to `source = stringAsset(...)` and writes the inline
// content to the destination on the remote.
func TestL2FileProvisionerContent(t *testing.T) {
	t.Parallel()
	installPulumiCommandPlugin(t)

	ssh := sshcontainer.Start(t)

	_, err := ssh.Exec("mkdir -p /tmp/conformance")
	require.NoError(t, err, "pre-creating remote dir")

	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Config: remoteExecConfig(ssh),
		Input: map[string]string{
			"main.tf": remoteExecConnectionVars + `
resource "test_resource" "example" {
  value = "hello"

  provisioner "file" {
    connection {
      host        = var.ssh_host
      port        = var.ssh_port
      user        = var.ssh_user
      private_key = var.ssh_private_key
    }
    content     = "inline content for ${var.conformance_kind}\n"
    destination = "/tmp/conformance/content-${var.conformance_kind}.txt"
  }
}

output "value" {
  value = test_resource.example.value
}
`,
		},
	})

	for _, kind := range []string{"tf", "pulumi"} {
		out, err := ssh.Exec("cat /tmp/conformance/content-" + kind + ".txt")
		require.NoError(t, err, "%s content file", kind)
		assert.Equal(t, "inline content for "+kind+"\n", out)
	}
}

// TestL2FileProvisionerResourceConnection verifies that a `file` provisioner
// inherits a `connection` block defined on the parent resource when the
// provisioner itself does not declare one.
func TestL2FileProvisionerResourceConnection(t *testing.T) {
	t.Parallel()
	installPulumiCommandPlugin(t)

	ssh := sshcontainer.Start(t)

	_, err := ssh.Exec("mkdir -p /tmp/conformance")
	require.NoError(t, err, "pre-creating remote dir")

	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Config: remoteExecConfig(ssh),
		Input: map[string]string{
			"main.tf": remoteExecConnectionVars + `
resource "test_resource" "example" {
  value = "hello"

  connection {
    host        = var.ssh_host
    port        = var.ssh_port
    user        = var.ssh_user
    private_key = var.ssh_private_key
  }

  provisioner "file" {
    content     = "from resource conn ${var.conformance_kind}\n"
    destination = "/tmp/conformance/resource-conn-${var.conformance_kind}.txt"
  }
}

output "value" {
  value = test_resource.example.value
}
`,
		},
	})

	for _, kind := range []string{"tf", "pulumi"} {
		out, err := ssh.Exec("cat /tmp/conformance/resource-conn-" + kind + ".txt")
		require.NoError(t, err, "%s file", kind)
		assert.Equal(t, "from resource conn "+kind+"\n", out)
	}
}

// TestL2FileProvisionerMultiple verifies that multiple `file` provisioners on a
// single resource convert to a chain of CopyToRemote resources where each
// depends on the previous (and the first depends on the parent resource), so
// they execute in the order Terraform would run them.
func TestL2FileProvisionerMultiple(t *testing.T) {
	t.Parallel()
	installPulumiCommandPlugin(t)

	ssh := sshcontainer.Start(t)

	_, err := ssh.Exec("mkdir -p /tmp/conformance")
	require.NoError(t, err, "pre-creating remote dir")

	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Config: remoteExecConfig(ssh),
		Input: map[string]string{
			"main.tf": remoteExecConnectionVars + `
resource "test_resource" "example" {
  value = "hello"

  connection {
    host        = var.ssh_host
    port        = var.ssh_port
    user        = var.ssh_user
    private_key = var.ssh_private_key
  }

  provisioner "file" {
    content     = "first ${var.conformance_kind}\n"
    destination = "/tmp/conformance/multi-first-${var.conformance_kind}.txt"
  }

  provisioner "file" {
    content     = "second ${var.conformance_kind}\n"
    destination = "/tmp/conformance/multi-second-${var.conformance_kind}.txt"
  }
}

output "value" {
  value = test_resource.example.value
}
`,
		},
	})

	for _, kind := range []string{"tf", "pulumi"} {
		out, err := ssh.Exec("cat /tmp/conformance/multi-first-" + kind + ".txt")
		require.NoError(t, err, "%s first", kind)
		assert.Equal(t, "first "+kind+"\n", out)

		out, err = ssh.Exec("cat /tmp/conformance/multi-second-" + kind + ".txt")
		require.NoError(t, err, "%s second", kind)
		assert.Equal(t, "second "+kind+"\n", out)
	}
}

// TestL2FileProvisionerProvisionerConnectionOverridesResource verifies that
// when both the resource and the provisioner declare a `connection` block, the
// provisioner-level connection wins. The resource-level block points at a host
// that is guaranteed not to resolve, so if the converter ever loses the
// override the Pulumi run would fail to connect.
func TestL2FileProvisionerProvisionerConnectionOverridesResource(t *testing.T) {
	t.Parallel()
	installPulumiCommandPlugin(t)

	ssh := sshcontainer.Start(t)

	_, err := ssh.Exec("mkdir -p /tmp/conformance")
	require.NoError(t, err, "pre-creating remote dir")

	conformance.AssertConversion(t, conformance.TestCase{
		Providers: []conformance.Provider{
			{Name: "test", Factory: providers.TestProvider},
		},
		Config: remoteExecConfig(ssh),
		Input: map[string]string{
			"main.tf": remoteExecConnectionVars + `
resource "test_resource" "example" {
  value = "hello"

  # Bogus resource-level connection. If the converter mistakenly used this
  # instead of the provisioner-level override, the Pulumi run would fail to
  # connect.
  connection {
    host        = "invalid.example.invalid"
    port        = 1
    user        = "nobody"
    private_key = "ignored"
  }

  provisioner "file" {
    connection {
      host        = var.ssh_host
      port        = var.ssh_port
      user        = var.ssh_user
      private_key = var.ssh_private_key
    }
    content     = "override ${var.conformance_kind}\n"
    destination = "/tmp/conformance/override-${var.conformance_kind}.txt"
  }
}

output "value" {
  value = test_resource.example.value
}
`,
		},
	})

	for _, kind := range []string{"tf", "pulumi"} {
		out, err := ssh.Exec("cat /tmp/conformance/override-" + kind + ".txt")
		require.NoError(t, err, "%s file", kind)
		assert.Equal(t, "override "+kind+"\n", out)
	}
}

// TestL2FileProvisionerSourceDynamic verifies that a `file` provisioner whose
// `source` is a non-literal expression (cannot be statically resolved at
// convert time) converts to `source = try(fileAsset(p), fileArchive(p))` —
// letting Pulumi pick the correct asset shape at runtime. Two sub-tests
// exercise both runtime branches of the `try`: one where the variable resolves
// to a file (fileAsset succeeds) and one where it resolves to a directory
// (fileAsset fails, fileArchive succeeds).
func TestL2FileProvisionerSourceDynamic(t *testing.T) {
	t.Parallel()
	installPulumiCommandPlugin(t)

	const tfProgram = `
variable "src_path" {
  type = string
}

resource "test_resource" "example" {
  value = "hello"

  provisioner "file" {
    connection {
      host        = var.ssh_host
      port        = var.ssh_port
      user        = var.ssh_user
      private_key = var.ssh_private_key
    }
    source      = var.src_path
    destination = "/tmp/conformance/dynamic-${var.conformance_kind}"
  }
}

output "value" {
  value = test_resource.example.value
}
`

	t.Run("file", func(t *testing.T) {
		t.Parallel()

		ssh := sshcontainer.Start(t)

		_, err := ssh.Exec("mkdir -p /tmp/conformance")
		require.NoError(t, err, "pre-creating remote dir")

		cfg := remoteExecConfig(ssh)
		cfg["src_path"] = "./hello.txt"

		conformance.AssertConversion(t, conformance.TestCase{
			Providers: []conformance.Provider{
				{Name: "test", Factory: providers.TestProvider},
			},
			Config: cfg,
			Input: map[string]string{
				"main.tf":   remoteExecConnectionVars + tfProgram,
				"hello.txt": "dynamic source\n",
			},
		})

		for _, kind := range []string{"tf", "pulumi"} {
			out, err := ssh.Exec("cat /tmp/conformance/dynamic-" + kind)
			require.NoError(t, err, "%s dynamic file", kind)
			assert.Equal(t, "dynamic source\n", out)
		}
	})

	t.Run("dir", func(t *testing.T) {
		t.Parallel()

		ssh := sshcontainer.Start(t)

		for _, kind := range []string{"tf", "pulumi"} {
			_, err := ssh.Exec("mkdir -p /tmp/conformance/dynamic-" + kind)
			require.NoError(t, err, "pre-creating %s remote dir", kind)
		}

		cfg := remoteExecConfig(ssh)
		cfg["src_path"] = "./payload/"

		conformance.AssertConversion(t, conformance.TestCase{
			Providers: []conformance.Provider{
				{Name: "test", Factory: providers.TestProvider},
			},
			Config: cfg,
			Input: map[string]string{
				"main.tf":       remoteExecConnectionVars + tfProgram,
				"payload/a.txt": "alpha\n",
				"payload/b.txt": "bravo\n",
			},
		})

		for _, kind := range []string{"tf", "pulumi"} {
			out, err := ssh.Exec("cat /tmp/conformance/dynamic-" + kind + "/a.txt")
			require.NoError(t, err, "%s a.txt", kind)
			assert.Equal(t, "alpha\n", out)

			out, err = ssh.Exec("cat /tmp/conformance/dynamic-" + kind + "/b.txt")
			require.NoError(t, err, "%s b.txt", kind)
			assert.Equal(t, "bravo\n", out)
		}
	})
}
