config "numberIn" "number" {
  description = "This is an example of a variable description"
}

config "stringIn" "string" {
}

config "nullableStringIn" "string" {
  default = null
}

config "optAnyIn" {
  default = null
}

config "anyWithDefault" {
  default = {}
}

config "boolIn" "bool" {
}

config "stringListIn" "list(optional(string))" {
}

config "stringMapIn" "map(optional(string))" {
}

config "stringMapAnyIn" "map(any)" {
}

config "objectIn" "object({first=number, second=string})" {
}
