config "aThing" {
}
myaThing = true

resource "aThingResource" "simple:index:resource" {
  __logicalName = "a_thing"
  inputOne      = "Hello ${aThing}"
  inputTwo      = myaThing
}

aThingDataSource = invoke("simple:index:dataSource", {
  inputOne = "Hello ${aThingResource.result}"
  inputTwo = myaThing
})

resource "aThingAnotherResource" "simple:index:anotherResource" {
  __logicalName = "a_thing"
  inputOne      = "Hello ${aThingResource.result}"
}

output "aThing" {
  value = aThingDataSource.result
}
