config "listvar" {
  default = ["2025"]
}

config "dynvar" "list(object({innerList=list(object({nestedValue=string})), innerValue=string}))" {
  default = []
}

resource "aResource" "blocks:index/index:resource" {
  __logicalName = "a_resource"
  options {
    range = length(listvar) > 0 ? 1 : 0
  }
  aListOfResources = [for entry in entries(dynvar) : {
    innerResources = [for entry2 in entries(entry.value.innerList) : {

      # Utilize the inner resource in the inner dynamic block, this
      # worked even before pulumi/pulumi#18718 was fixed.
      nestedString = entry2.value.nestedValue
    }]
    innerString = entry.value.innerValue != null ? "TrySuccess" : "TryFail"
  }]
}

resource "bResource" "blocks:index/index:resource" {
  __logicalName = "b_resource"
  options {
    range = length(listvar) > 0 ? 1 : 0
  }
  aListOfResources = [for entry in entries(dynvar) : {
    innerResources = [for entry2 in entries(entry.value.innerList) : {

      # This was fixed by pulumi/pulumi#18718.  Before the generated
      # PCL would shadow a_list_of_resources with the same iterator
      # name (entry). 
      nestedString = entry.value.innerValue
    }]
    innerString = entry.value.innerValue != null ? "TrySuccess" : "TryFail"
  }]
}
