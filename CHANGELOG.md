CHANGELOG
=========

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