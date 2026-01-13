### Improvements

 - Handle implicitly required providers that are non-Pulumi bridged providers by resolving and parameterizing them using their latest version from the registry.

### Bug Fixes

 - Fix casing mismatch in lookup function by preserving original key casing in map arguments.