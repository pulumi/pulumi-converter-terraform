CHANGELOG
=========

## 1.0.22

- Bump generated provider-terraform version

## 1.0.21

### Improvements

- Change inferred resource names to pascal case
- Add parameterization block to "package" blocks
- Add generation of pcl "package" blocks

## 1.0.20

### Improvements

- Add EOT (heredoc) style string delimiter handling.
- Add template join expression to convert expression
- Add references to issues for missing functions in output
- Add code generation rename workarounds for pcl keywords

### Bug Fixes

- Fix errors being encountered but not reported to the user

- Fix using a module multiple times via different constraints

- Fix conversion of object blocks

- Fix conversion of object attributes

## 1.0.18

### Improvements

- Support custom translation of appautoscaling ids

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
