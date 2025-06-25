// Copyright 2016-2023, Pulumi Corporation.
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

package tests

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type parsedExample struct {
	cloneURL string
	org      string
	repo     string
	path     string
}

func parseExample(t *testing.T, v string) parsedExample {
	t.Helper()
	require.True(t, strings.HasPrefix(v, "https://github.com/"))
	trimmed := strings.TrimPrefix(v, "https://github.com/")
	segments := strings.Split(trimmed, "/")
	require.True(t, len(segments) >= 2)

	var path string
	if len(segments) > 2 {
		path = filepath.Join(segments[2:]...)
	}

	return parsedExample{
		cloneURL: fmt.Sprintf("https://github.com/%s/%s.git", segments[0], segments[1]),
		org:      segments[0],
		repo:     segments[1],
		path:     path,
	}
}

type keyedMutex struct {
	mutexes sync.Map
}

func (m *keyedMutex) Lock(key string) func() {
	value, _ := m.mutexes.LoadOrStore(key, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return func() { mutex.Unlock() }
}

type stringSet map[string]struct{}

func newStringSet(values ...string) stringSet {
	s := stringSet{}
	for _, v := range values {
		s[v] = struct{}{}
	}
	return s
}

func (ss stringSet) Has(s string) bool {
	_, ok := ss[s]
	return ok
}

func (ss stringSet) Equal(other stringSet) bool {
	if len(ss) != len(other) {
		return false
	}
	for k := range ss {
		if !other.Has(k) {
			return false
		}
	}
	return true
}

const (
	csharp     = "c#"
	golang     = "go"
	python     = "python"
	typescript = "typescript"
)

var allLanguages = newStringSet(csharp, golang, python, typescript)

func TestExample(t *testing.T) {
	t.Parallel()

	km := keyedMutex{}

	languages := []string{
		csharp,
		golang,
		python,
		typescript,
	}

	tests := []struct {
		example string
		strict  bool
		commit  string
		skip    stringSet

		testOnlyThisExampleInLanguage string
	}{
		{
			example: "https://github.com/kube-hetzner/terraform-hcloud-kube-hetzner",
			// TODO[pulumi/pulumi-converter-terraform#212]: Hetzner cloud has various conversion issues.
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-security-group/examples/complete",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc",
			// TODO[pulumi/pulumi#13743]: unknown property 'domain' among [...]
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc/examples/complete",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc/examples/ipam",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc/examples/ipv6-dualstack",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc/examples/ipv6-only",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc/examples/issues",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc/examples/manage-default-vpc",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc/examples/network-acls",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc/examples/outpost",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc/examples/secondary-cidr-blocks",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc/examples/separate-route-tables",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc/examples/simple",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-vpc/examples/vpc-flow-logs",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-account",
			strict:  true,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-assumable-role",
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-assumable-role-with-oidc",
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-assumable-role-with-saml",
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-assumable-roles",
			strict:  true,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-assumable-roles-with-saml",
			strict:  true,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-eks-role",
			// TODO[pulumi/pulumi-converter-terraform#32]: upstream example change can no longer convert
			// Was incidentally broken by https://github.com/pulumi/pulumi-converter-terraform/pull/91.
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-github-oidc",
			strict:  true,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-group-complete",
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-group-with-assumable-roles-policy",
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-group-with-policies",
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-policy",
			strict:  true,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-read-only-policy",
			// TODO(bpollack) this started failing in CI but seems to pass with
			// pulumi/pulumi changes which are blocked on this package's release.
			// Disabling temporarily.
			skip:   stringSet{csharp: struct{}{}},
			strict: true,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-role-for-service-accounts-eks",
			// TODO[pulumi/pulumi-converter-terraform#200]: caused by error converting aws vpc module.
			// TODO[pulumi/pulumi-converter-terraform#32]: upstream example change can no longer convert
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-iam/examples/iam-user",
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-security-group",
			strict:  true,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-security-group/examples/computed",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-security-group/examples/disabled",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-security-group/examples/dynamic",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-security-group/examples/http",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-security-group/examples/rules-only",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-eks",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-lambda",
			// TODO[pulumi/pulumi-converter-terraform#104]: `unsupported attribute 'timeouts'` error as of v6.7.1
			commit: "a729331518fec8adf232e9a2ad520a5bbc815b26", // v6.7.0
			strict: true,
		},
		{
			example:                       "https://github.com/terraform-aws-modules/terraform-aws-s3-bucket",
			strict:                        true,
			testOnlyThisExampleInLanguage: "typescript",
			commit:                        "f90d8a385e4c70afd048e8997dcccf125b362236",
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-s3-bucket/examples/object",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip:   allLanguages,
			commit: "f90d8a385e4c70afd048e8997dcccf125b362236",
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-s3-bucket/examples/complete",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip:   allLanguages,
			commit: "f90d8a385e4c70afd048e8997dcccf125b362236",
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-rds",
			// TODO[pulumi/pulumi#18446 strict should work if the plugin is available (std in this case).
			// strict:  true,
			// TODO[pulumi/pulumi#18448 when std is required for go conversion fails.
			skip: newStringSet(golang),
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-alb",
			strict:  true,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-alb/examples/complete-alb",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-alb/examples/complete-nlb",
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-rds-aurora",
			// Pin to the commit associated with v9.0.2 of the module. The most recent release (v9.1.0) uses
			// new attributes on resource/aws_rds_cluster (domain and domain_iam_role_name) which were included
			// in v5.37.0 of the TF AWS provider, but the Pulumi AWS provider has not yet been updated to use
			// this version of the TF AWS provider.
			// https://github.com/terraform-aws-modules/terraform-aws-rds-aurora/releases/tag/v9.0.2
			// https://github.com/terraform-aws-modules/terraform-aws-rds-aurora/releases/tag/v9.1.0
			// https://github.com/hashicorp/terraform-provider-aws/releases/tag/v5.37.0
			// https://github.com/pulumi/pulumi-aws/releases/tag/v6.22.0
			commit: "1b34843f9ffeef885ef24edb3f87336ad9daf9d2",
			strict: true,
		},
		{
			example: "https://github.com/terraform-aws-modules/terraform-aws-acm",
			strict:  true,
			// TODO[pulumi/pulumi-converter-terraform#32]: upstream example change can no longer convert
			skip: allLanguages,
		},
		{
			example: "https://github.com/avantoss/vault-infra/terraform/main",
			strict:  true,
		},
		{
			example: "https://github.com/philips-labs/terraform-aws-github-runner",
			strict:  true,
			// TODO[pulumi/pulumi-converter-terraform#21]: Crashes in CI (uses too many resources?)
			skip: allLanguages,
		},
		{
			example: "https://github.com/aztfmod/terraform-azurerm-caf",
			// TODO[pulumi/pulumi-converter-terraform#186]: Should use terraform bridge, error details in pulumi/pulumi-terraform-bridge#205
			// TODO[pulumi/pulumi-converter-terraform#206]: missing attributes vcores and clientConfig
			// TODO[pulumi/pulumi-terraform-bridge#1303]: panic: fatal: An assertion has failed: empty path part passed into getInfo: .recurrence.hours
			// TODO[pulumi/pulumi-converter-terraform#112]:  empty path part passed into getInfo: .recurrence.hours (same as above)
			skip: allLanguages,
		},
		{
			example: "https://github.com/awslabs/data-on-eks/analytics/terraform/spark-k8s-operator",
			// TODO[pulumi/pulumi#13581]: circular reference
			skip: allLanguages,
		},
		{
			example: "https://github.com/aws-samples/hub-and-spoke-with-inspection-vpc-terraform",
			// TODO[pulumi/pulumi#13581]: circular reference
			skip: allLanguages,
		},
		{
			example: "https://github.com/aws-ia/terraform-aws-eks-blueprints/patterns/multi-tenancy-with-teams",
			// TODO[pulumi/pulumi-converter-terraform#200]: caused by error converting aws vpc module.
			skip: allLanguages,
		},
	}

	// There could be more than one example in a repo. To keep things simple for now,
	// they all must have the same commit specified. Verify that here.
	commits := map[string]string{}
	for _, tt := range tests {
		parsed := parseExample(t, tt.example)
		if commit, ok := commits[parsed.cloneURL]; ok {
			require.Equal(t, commit, tt.commit, "all examples in %q must use the same commit", parsed.cloneURL)
		} else {
			commits[parsed.cloneURL] = tt.commit
		}
	}

	hasNarrowing := false
	for _, tt := range tests {
		if tt.testOnlyThisExampleInLanguage != "" {
			hasNarrowing = true
		}
	}

	for _, tt := range tests {
		t.Run(tt.example, func(t *testing.T) {
			t.Parallel()

			if hasNarrowing && tt.testOnlyThisExampleInLanguage == "" {
				t.Skip()
			}

			if tt.skip.Equal(allLanguages) {
				t.Skip()
			}

			parsed := parseExample(t, tt.example)

			orgDir := filepath.Join("repos", parsed.org)
			require.NoError(t, os.MkdirAll(orgDir, 0o700), "creating repo org directory")
			repoDir := filepath.Join(orgDir, parsed.repo)

			// Clone the repo locally, if it doesn't already exist.
			unlock := km.Lock(repoDir)
			if _, err := os.Stat(repoDir); os.IsNotExist(err) {
				_, _, err = runCommand(t, orgDir, "git", "clone", "--depth", "1", parsed.cloneURL)
				require.NoError(t, err, "cloning repo")
			}

			// If we have a specific commit, use that.
			if tt.commit != "" {
				_, _, err := runCommand(t, repoDir, "git", "fetch", "--depth", "1", "origin", tt.commit)
				require.NoError(t, err, "fetching commit: %s", tt.commit)
				_, _, err = runCommand(t, repoDir, "git", "checkout", tt.commit)
				require.NoError(t, err, "checking out commit: %s", tt.commit)
			}
			unlock()

			// Test each language.
			exampleDir := filepath.Join(repoDir, parsed.path)
			for _, language := range languages {
				language := language
				t.Run(language, func(t *testing.T) {
					t.Parallel()

					if hasNarrowing && tt.testOnlyThisExampleInLanguage != language {
						t.Skip()
					}

					if tt.skip.Has(language) {
						t.Skip()
					}

					testExample(t, exampleDir, language, tt.strict)
				})
			}
		})
	}
}

func testExample(t *testing.T, path, language string, strict bool) {
	outputDir, err := os.MkdirTemp("", "converter-output")
	require.NoError(t, err, "creating temp directory for test")
	defer func() {
		if !t.Failed() {
			err := os.RemoveAll(outputDir)
			require.NoErrorf(t, err, "cleaning up temp test directory %q", outputDir)
		}
	}()

	args := []string{
		"convert",
		"--generate-only",
		"--from", "terraform",
		"--language", language,
		"--out", outputDir,
	}
	if strict {
		args = append(args, "--strict")
	}

	stdout, stderr, err := runCommand(t, path, "pulumi", args...)
	if err != nil {
		t.Logf("Command failed: %s", err)
		t.Logf("STDOUT: %s", stdout)
		t.Logf("STDERR: %s", stderr)
		t.FailNow()
	}

	if language == "typescript" {
		logNotImplementedReport(t, outputDir, ".ts")
	}
}

func runCommand(t *testing.T, cwd, command string, args ...string) (string, string, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	cmd := exec.Command(command, args...)
	cmd.Dir = cwd
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	t.Logf("Running command: %s %s", command, strings.Join(args, " "))
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func logNotImplementedReport(t *testing.T, path, extension string) {
	report := notImplementedReport(t, path, extension)

	type Pair struct {
		Key   string
		Value int
	}

	pairs := make([]Pair, len(report))
	i := 0
	for k, v := range report {
		pairs[i] = Pair{k, v}
		i++
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})

	var total int
	for _, p := range pairs {
		total += p.Value
	}
	if total > 0 {
		t.Logf("%d total notImplemented", total)
	}
	for _, p := range pairs {
		t.Logf("notImplemented (%v): %s\n", p.Value, p.Key)
	}
}

func notImplementedReport(t *testing.T, path, extension string) map[string]int {
	result := make(map[string]int)

	regex := regexp.MustCompile(`(?mU)notImplemented\([\x60"](.*)[\x60"(]`)

	files, err := filepath.Glob(filepath.Join(path, "*"+extension))
	require.NoError(t, err, "globbing files")

	for _, file := range files {
		contents, err := os.ReadFile(file)
		require.NoError(t, err, "reading file %q", file)

		matches := regex.FindAllStringSubmatch(string(contents), -1)
		if len(matches) > 0 {
			for _, match := range matches {
				result[match[1]]++
			}
		}
	}

	return result
}
