output "nullOut" {
  value = null
}

output "numberOut" {
  value = 0
}

output "boolOut" {
  value = true
}

output "stringOut" {
  value = "hello world"
}

output "tupleOut" {
  value = [1, 2, 3]
}

output "numberOperatorsOut" {
  value = -(1 + 2) * 3 / 4 % 5
}

output "boolOperatorsOut" {
  value = !(true || false) && true
}

output "strObjectOut" {
  value = {
    hello   = "hallo"
    goodbye = "ha det"
  }
}

output "sortedObjectOut" {
  value = {
    nested = {
      b = 4
      a = 3
    }
    b = 2
    a = 1
  }
}
a_key   = "hello"
a_value = -1
a_list  = [1, 2, 3]
a_list_of_maps = [{
  x = [1, 2]
  y = [3, 4]
  }, {
  x = [5, 6]
  y = [7, 8]
}]

output "staticIndexOut" {
  value = a_list[1]
}

output "dynamicIndexOut" {
  value = a_list[a_value]
}

output "complexObjectOut" {
  value = {
    aTuple = ["a", "b", "c"]
    anObject = {
      literalKey                = 1
      anotherLiteralKey         = 2
      "yet_another_literal_key" = a_value

      // This only translates correctly in the new converter.
      (a_key) = 4
    }
    ambiguousFor = {
      "for" = 1
    }
  }
}

output "simpleTemplate" {
  value = "${a_value}"
}

output "quotedTemplate" {
  value = "The key is ${a_key}"
}

output "heredoc" {
  value = "This is also a template.\nSo we can output the key again ${a_key}\n"
}

output "forTuple" {
  value = [for key, value in ["a", "b"] : "${key}:${value}:${a_value}" if key != 0]
}

output "forTupleValueOnly" {
  value = [for value in ["a", "b"] : "${value}:${a_value}"]
}

output "forTupleValueOnlyAttr" {
  value = [for x in [{
    id   = "i-123"
    zone = "us-west"
    }, {
    id   = "i-abc"
    zone = "us-east"
  }] : x.id if x.zone == "us-east"]
}

output "forObject" {
  value = { for key, value in ["a", "b"] : key => "${value}:${a_value}" if key != 0 }
}

output "forObjectGrouping" {
  value = { for key, value in ["a", "a", "b"] : key => value... if key > 0 }
}

output "relativeTraversalAttr" {
  value = a_list_of_maps[0].x
}

output "relativeTraversalIndex" {
  value = a_list_of_maps[0]["x"]
}

output "conditionalExpr" {
  value = a_value == 0 ? "true" : "false"
}
