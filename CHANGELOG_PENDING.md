### Improvements

 - Handle implicitly required providers that are non-Pulumi bridged providers by resolving and parameterizing them using their latest version from the registry.

### Bug Fixes

 - Stop emitting deprecated package block labels in generated PCL, using the `baseProviderName` attribute instead.