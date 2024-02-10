config "aThing" {
}
myaThing = true

resource "aThingResource" "simple:index:resource" {
  __logicalName = "a_thing"
  inputOne      = "Hello ${aThing}"
  inputTwo      = myaThing
}

aThingData = invoke("simple:index:dataSource", {
  inputOne = "Hello ${aThingResource.result}"
  inputTwo = myaThing
})

output "aThing" {
  value = aThingData.result
}
