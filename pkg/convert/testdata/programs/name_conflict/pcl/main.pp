config "aThing" {
}
myAThing = true

resource "aThingResource" "simple:index:resource" {
  __logicalName = "a_thing"
  inputOne      = "Hello ${aThing}"
  inputTwo      = myAThing
}

aThingDataSource = invoke("simple:index:dataSource", {
  inputOne = "Hello ${aThingResource.result}"
  inputTwo = myAThing
})

resource "aThingAnotherResource" "simple:index:anotherResource" {
  __logicalName = "a_thing"
  inputOne      = "Hello ${aThingResource.result}"
}

output "aThing" {
  value = aThingDataSource.result
}
