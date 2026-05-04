### Improvements

- Handle implicitly required providers that are non-Pulumi bridged providers by resolving and parameterizing them using
   their latest version from the registry.
   [#371](https://github.com/pulumi/pulumi-converter-terraform/issues/371)
   
- Support converting the `try` TF intrinsic to PCL.
   [#16](https://github.com/pulumi/pulumi-converter-terraform/issues/16)

- Convert `lifecycle.ignore_changes` to `ignoreChanges` resource option in generated Pulumi code.
  [#60](https://github.com/pulumi/pulumi-converter-terraform/issues/60)

- Convert `remote-exec` provisioners to `command:remote:Command`. The `inline` form
  emits a single Command resource; `script` adds a paired `command:remote:CopyToRemote`
  to upload the script before invoking it; `scripts` parallelizes the upload via
  `range` and runs each script sequentially. The TF `connection` block is mapped to
  the Pulumi `Connection` input, including `bastion_*` fields which become a `proxy`
  sub-object. The `connection.timeout` attribute is propagated to a
  `customTimeouts` resource option (`create` and `update`) on every generated
  Command/CopyToRemote.
  [#430](https://github.com/pulumi/pulumi-converter-terraform/issues/430)

### Bug Fixes

- Fix dynamic blocks with list-typed `for_each` incorrectly wrapping the collection in `entries()`.
  [#414](https://github.com/pulumi/pulumi-converter-terraform/issues/414)

- Fix `self.X` references inside `provisioner` blocks being converted to an undefined variable. They
  now resolve to the Pulumi-renamed attribute on the parent resource.
