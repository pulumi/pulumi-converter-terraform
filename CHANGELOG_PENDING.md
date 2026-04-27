### Improvements

- Handle implicitly required providers that are non-Pulumi bridged providers by resolving and parameterizing them using
   their latest version from the registry.
   [#371](https://github.com/pulumi/pulumi-converter-terraform/issues/371)
   
- Support converting the `try` TF intrinsic to PCL.
   [#16](https://github.com/pulumi/pulumi-converter-terraform/issues/16)

- Convert `lifecycle.ignore_changes` to `ignoreChanges` resource option in generated Pulumi code.
  [#60](https://github.com/pulumi/pulumi-converter-terraform/issues/60)

- Convert the `remote-exec` provisioner to a `command:remote:Command` resource. `inline` becomes a
  runtime `join("\n", ...)`; `script` and `scripts` become one or more `command:remote:CopyToRemote`
  resources (with `fileAsset` sources) chained before the `command:remote:Command` that runs them.
  When upload resources are emitted, the connection is hoisted to a top-level local so the upload
  and run resources share it.
  [#430](https://github.com/pulumi/pulumi-converter-terraform/issues/430)

### Bug Fixes

- Fix dynamic blocks with list-typed `for_each` incorrectly wrapping the collection in `entries()`.
  [#414](https://github.com/pulumi/pulumi-converter-terraform/issues/414)

- Fix `self.X` references inside `provisioner` blocks being converted to an undefined variable. They
  now resolve to the Pulumi-renamed attribute on the parent resource.
