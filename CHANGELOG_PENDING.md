### Improvements

 - Handle implicitly required providers that are non-Pulumi bridged providers by resolving and parameterizing them using their latest version from the registry.

### Bug Fixes

 - Fix camelCase conversion for string-indexed object properties. Bracket notation like `each.value["aws_region"]` now correctly converts to `range.value["awsRegion"]`.