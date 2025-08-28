values = {
  something = {
    someKey = "some-value"
  }
  "something-else" = {
    someKey = "some-value"
  }
}
resource "aResourceWithForeachMap" "simple:index:resource" {
  __logicalName = "a_resource_with_foreach_map"
  options {
    range = {
      cruel = "world"
      good  = "class"
    }
  }
  inputOne = "Hello ${range.key} ${range.value}"
  inputTwo = 0
}

output "someOutputA" {
  value = aResourceWithForeachMap["cruel"].result
}

aDataSourceWithForeachMap = { for __key, __value in {
  cruel = "world"
  good  = "class"
  } : __key => invoke("simple:index:dataSource", {
    inputOne = "Hello ${__key} ${__value}"
    inputTwo = true
}) }

output "someOutputB" {
  value = aDataSourceWithForeachMap["cruel"].result
}

resource "aResourceWithForeachArray" "simple:index:resource" {
  __logicalName = "a_resource_with_foreach_array"
  options {
    range = ["cruel", "good"]
  }
  inputOne = "Hello ${range.value} world"
  inputTwo = true
}

resource "aResourceWithForeachObjectAccess" "simple:index:resource" {
  __logicalName = "a_resource_with_foreach_object_access"
  options {
    range = values
  }
  inputOne = range.value.someKey
}

resource "aResourceWithForeachObjectIndex" "simple:index:resource" {
  __logicalName = "a_resource_with_foreach_object_index"
  options {
    range = values
  }
  inputOne = range.value.someKey
}

output "someOutputC" {
  value = aResourceWithForeachArray["good"].result
}

aDataSourceWithForeachArray = { for __key, __value in ["cruel", "good"] : __key => invoke("simple:index:dataSource", {
  inputOne = "Hello ${__value} world"
  inputTwo = true
}) }

output "someOutputD" {
  value = aDataSourceWithForeachArray["good"].result
}
