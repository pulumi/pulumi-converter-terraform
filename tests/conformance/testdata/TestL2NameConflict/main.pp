config "aThing" {
}
myAThing = true

resource "aThingResource" "simple:index/resource:Resource" {
  __logicalName = "a_thing"
  inputOne      = "Hello ${aThing}"
  inputTwo      = myAThing
}

aThingGetDataSource = invoke("simple:index/getDataSource:getDataSource", {
  inputOne = "Hello ${aThingResource.result}"
  inputTwo = myAThing
})

resource "aThingAnotherResource" "simple:index/anotherResource:AnotherResource" {
  __logicalName = "a_thing"
  inputOne      = "Hello ${aThingResource.result}"
}

output "aThing" {
  value = aThingGetDataSource.result
}
