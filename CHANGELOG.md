CHANGELOG
=========

## 1.0.17

### Improvements

- Support the standard `tolist` function translation by the rewrite rule `tolist(x) ==> x`
- Don't attempt to rename object keys if the object contains any non-identifiers

## 1.0.16

### Improvements

- Support the `depends_on` option
- Emit warnings when encountering non-supported lifecycle hooks `create_before_destroy` and `replace_triggered_by`

### Bug Fixes

- Convert expressions inside `jsonencode` calls without rewriting object keys to camelCase

## 1.0.15

### Improvements

 - Ensure block and attribute iteration is consistent.

## 1.0.14

### Improvements

 - Sort generated properties by the position of their keys to get deterministic output.

## 1.0.13

### Improvements

  - Better conflict resolution for conflicting source names. They'll now use the objects type name to help build a unique name.

## 1.0.12

### Improvements

  - Support the `replace` function.

  - Don't rename map keys.

### Bug Fixes

 - Fix order of parameters for the `parseint` function