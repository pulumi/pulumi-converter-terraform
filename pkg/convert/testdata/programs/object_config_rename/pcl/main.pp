config "simpleObjectConfig" "object({first_member=number, second_member=string})" {
  default = {
    first_member  = 10
    second_member = "hello"
  }
}

config "objectListConfig" "list(object({first_member=number, second_member=string}))" {
  default = [{
    first_member  = 10
    second_member = "hello"
  }]
}

config "objectListConfigEmpty" "list(object({first_member=number, second_member=string}))" {
  default = []
}

config "objectMapConfig" "map(object({first_member=number, second_member=string}))" {
  default = {
    hello = {
      first_member  = 10
      second_member = "hello"
    }
  }
}

config "objectMapConfigEmpty" "map(object({first_member=number, second_member=string}))" {
  default = {}
}

resource "usingSimpleObjectConfig" "simple:index:resource" {
  __logicalName = "using_simple_object_config"
  inputOne      = simpleObjectConfig.first_member
}

resource "usingListObjectConfig" "simple:index:resource" {
  __logicalName = "using_list_object_config"
  inputOne      = objectListConfig[0].first_member
}

resource "usingListObjectConfigForEach" "simple:index:resource" {
  __logicalName = "using_list_object_config_for_each"
  options {
    range = objectListConfig
  }
  inputOne = range.value.first_member
}

resource "usingMapObjectConfig" "simple:index:resource" {
  __logicalName = "using_map_object_config"
  inputOne      = objectMapConfig["hello"].first_member
}

resource "usingMapObjectConfigForEach" "simple:index:resource" {
  __logicalName = "using_map_object_config_for_each"
  options {
    range = objectMapConfig
  }
  inputOne = range.value.first_member
}

resource "usingDynamic" "blocks:index/index:resource" {
  __logicalName = "using_dynamic"
  aListOfResources = [for entry in entries(objectMapConfig) : {
    innerString = entry.value.first_member
  }]
}

resource "usingDynamicIterator" "blocks:index/index:resource" {
  __logicalName = "using_dynamic_iterator"
  aListOfResources = [for entry in entries(objectMapConfig) : {
    innerString = entry.value.first_member
  }]
}
