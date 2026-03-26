# Converter conformance tests

Conformance tests verify that converting HCL programs through the Terraform
converter produces correct PCL and that the resulting Pulumi programs produce
the same outputs as running the original HCL directly against a Terraform
provider.

Each test runs two paths in parallel and asserts output equality:

- **Path A:** TF provider + HCL → `terraform apply` → outputs
- **Path B:** TF provider + HCL → convert to PCL → bridge provider → `pulumi up` → outputs

The generated PCL is also snapshot-tested against golden files in
`testdata/<TestName>/main.pp`. Set `PULUMI_ACCEPT=1` to update snapshots.

## Test levels

Tests are categorized by the complexity of the language features they exercise,
borrowing the level system from
[pulumi-test-language](https://github.com/pulumi/pulumi/tree/master/pkg/testing/pulumi-test-language):

- **L1 tests** do *not* exercise provider code paths and use only the most
  basic features (e.g. stack outputs, literals, expressions).
- **L2 tests** *do* exercise provider code paths and use things such as custom
  resources, function invocations, computed outputs, and `for_each`.

## Layout

```
tests/conformance/
├── README.md
├── providers/          # Shared test provider definitions
│   └── test.go
├── testdata/           # Golden PCL files, one directory per test
│   └── TestL2BasicResource/
│       └── main.pp
├── l2_basic_resource_test.go
└── l2_for_each_string_key_test.go
```

Each test lives in its own `_test.go` file. Shared provider schemas are defined
in the `providers` subpackage so they can be reused across tests.

## Running

```bash
go test ./tests/conformance/ -v -count=1 -timeout 120s
```

To update golden files:

```bash
PULUMI_ACCEPT=1 go test ./tests/conformance/ -v -count=1 -timeout 120s
```
