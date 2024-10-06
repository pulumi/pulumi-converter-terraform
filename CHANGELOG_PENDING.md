### Improvements

- Handle implicitly required providers that are non-Pulumi bridged providers by resolving and parameterizing them using
   their latest version from the registry.
   [#371](https://github.com/pulumi/pulumi-converter-terraform/issues/371)
   
- Support converting the `try` TF intrinsic to PCL.
   [#16](https://github.com/pulumi/pulumi-converter-terraform/issues/16)

- Convert `lifecycle.ignore_changes` to `ignoreChanges` resource option in generated Pulumi code.
  [#60](https://github.com/pulumi/pulumi-converter-terraform/issues/60)
- Implemented specialized conversion for `helm_release` resources.

### Bug Fixes

- Fix dynamic blocks with list-typed `for_each` incorrectly wrapping the collection in `entries()`.
  [#414](https://github.com/pulumi/pulumi-converter-terraform/issues/414)
