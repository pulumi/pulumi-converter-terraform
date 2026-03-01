items = {
  foo = {
    awsRegion    = "us-east-1"
    instanceType = "t2.micro"
  }
  bar = {
    awsRegion    = "us-west-2"
    instanceType = "t2.small"
  }
}


# Test 1: Direct string index on locals with bracket notation
output "localStringIndex" {
  value = items["foo"]["awsRegion"]
}


# Test 2: Using each.value with bracket notation in for_each
resource "testForEach" "simple:index:resource" {
  __logicalName = "test_for_each"
  options {
    range = items
  }
  inputOne = range.value["awsRegion"]
  inputTwo = 0
}


# Test 3: Using range.value with bracket notation in for expression
output "forExprStringIndex" {
  value = { for key, value in items : key => value["instanceType"] }
}
