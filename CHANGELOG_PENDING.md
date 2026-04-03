### Improvements

 - Handle implicitly required providers that are non-Pulumi bridged providers by resolving and parameterizing them using their latest version from the registry.

### Bug Fixes

 - Convert `lifecycle.ignore_changes` to `ignoreChanges` resource option in generated Pulumi code.