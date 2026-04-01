---
name: add-conformance-test
description: Run conformance tests for coverage, find a gap, and add a new targeted conformance test.
---

# Add a conformance test

Follow these steps exactly. Do not skip steps or combine them.

## Step 1: Run conformance tests with coverage

```bash
go test -count=1 -coverprofile=conformance_coverage.out -coverpkg=./pkg/convert/... -timeout 2h ./tests/conformance/
```

## Step 2: Identify coverage gaps

Run:

```bash
go tool cover -func=conformance_coverage.out | grep -v "100.0%" | grep -v "0.0%" | awk '{print $NF, $0}' | sort -g | cut -d' ' -f2-
```

Pick a function (or set of related functions) at 0% or low coverage that would benefit from
a conformance test. Read the function source to understand what HCL input would exercise it.

## Step 3: Decide the test level

- **L1** tests do NOT use any provider. They test variables, outputs, locals, expressions,
  and built-in functions. The `Providers` field of `conformance.TestCase` is left nil/empty.
- **L2** tests use one or more providers from `tests/conformance/providers/`. They test
  resource creation, data sources, and provider-dependent features.

Prefer L1 tests when possible -- they are simpler and faster.

## Step 4: Check that the harness can test the target behavior

Before writing the test, determine whether the harness can verify the behavior you want
to test. The harness verifies:

1. **Golden PCL match** -- the generated PCL text is compared against a snapshot file.
2. **Output value equality** -- `terraform apply` outputs are compared to `pulumi up` outputs.
3. **State assertions** -- the optional `AssertState` callback receives `[]apitype.ResourceV3`
   from the Pulumi deployment, allowing assertions on resource options like `Dependencies`,
   `IgnoreChanges`, `Protect`, `CustomTimeouts`, `ReplaceOnChanges`, etc.

If the behavior you want to test cannot be verified by any of these three mechanisms
(e.g. it requires multi-step updates, drift detection, or import), **stop and report
this to the user**. Do not write a test that cannot meaningfully verify the target behavior.

## Step 5: Write the test

Create a new file at `tests/conformance/<level>_<name>_test.go`. Follow the existing
patterns exactly:

```go
package conformance

import (
    "testing"
    "github.com/pulumi/pulumi-converter-terraform/pkg/testing/conformance"
)

func Test<Level><Name>(t *testing.T) {
    t.Parallel()
    conformance.AssertConversion(t, conformance.TestCase{
        // For L2 tests, include Providers:
        // Providers: []conformance.Provider{
        //     {Name: "test", Factory: providers.TestProvider},
        // },
        //
        // For tests with terraform variables, include Config:
        // Config: map[string]string{
        //     "key": "value",
        // },
        //
        // To assert on resource state (dependencies, ignoreChanges, etc.):
        // AssertState: func(t *testing.T, resources []apitype.ResourceV3) {
        //     t.Helper()
        //     res := findResource(resources, "myResource")
        //     assert.Contains(t, res.Dependencies, otherRes.URN)
        //     assert.Equal(t, []string{"value"}, res.IgnoreChanges)
        // },
        HCL: `
// your HCL here
`,
    })
}
```

If the existing test provider in `tests/conformance/providers/test.go` does not have the
schema needed for your test, extend it or add a new provider file. Do NOT modify the
harness itself.

## Step 6: Generate the golden file and run

```bash
PULUMI_ACCEPT=1 go test -v -count=1 -timeout 2h -run <TestName> ./tests/conformance/
```

The golden file at `tests/conformance/testdata/<TestName>/main.pp` is generated
automatically.

If the test fails, determine whether the failure is a genuine converter bug (e.g. incorrect
PCL output, wrong runtime values, missing resource options in state) or a test authoring
mistake.

- **Test authoring mistake**: fix the test and re-run.
- **Genuine converter bug**: the test is still valuable -- it documents the bug. Add a
  `t.Skip(...)` call before `t.Parallel()` with a message describing the failure. If there
  is an existing GitHub issue in `pulumi/pulumi-converter-terraform` that matches, link it
  in the skip message. If no issue exists, tell the user they should create one and leave a
  TODO in the skip message with a placeholder. Example:

  ```go
  // TODO[pulumi/pulumi-converter-terraform#NNN]: description
  t.Skip("description of the converter bug")
  ```

  Delete the golden file directory if one was created, since a skipped test should not
  have golden files that may go stale.

**Important**: Do NOT preemptively skip a test. Only add `t.Skip` after you have run the
test and confirmed it fails due to a converter bug. If you are unsure whether the test
will fail, run it first.

## Step 7: Run without PULUMI_ACCEPT to confirm golden match

```bash
go test -v -count=1 -timeout 2h -run <TestName> ./tests/conformance/
```

Skip this step if the test is skipped due to a converter bug.

## Step 8: Verify the full conformance suite still passes

```bash
go test -v -count=1 -timeout 2h ./tests/conformance/
```

## Step 9: Measure the coverage change

```bash
go test -count=1 -coverprofile=conformance_coverage_new.out -coverpkg=./pkg/convert/... -timeout 2h ./tests/conformance/
```

Compare the target function's coverage before and after:

```bash
go tool cover -func=conformance_coverage.out | grep <function_name>
go tool cover -func=conformance_coverage_new.out | grep <function_name>
```

Report the per-function and total coverage delta to the user.
