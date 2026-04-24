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

package tfexec

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

func (d *Driver) execTf(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command(getTFCommand(), args...) //nolint:gosec // args are test-controlled
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = d.cwd
	cmd.Env = os.Environ()
	if reattach := d.formatReattachEnvVar(); reattach != "" {
		cmd.Env = append(cmd.Env, reattach)
	}
	for k, v := range d.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	t.Logf("%s", cmd.String())
	err := cmd.Run()
	if err != nil {
		t.Logf("error from %q\n\nStdout:\n%s\n\nStderr:\n%s\n\n",
			cmd.String(), stdout.String(), stderr.String())
	}
	if stderrStr := stderr.String(); len(stderrStr) > 0 {
		t.Logf("%q stderr:\n%s\n", cmd.String(), stderrStr)
	}
	return stdout.Bytes(), err
}
