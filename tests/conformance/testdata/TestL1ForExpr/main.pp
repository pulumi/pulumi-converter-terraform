config "names" "list(string)" {
}

config "labels" "map(string)" {
}

output "upperNames" {
  value = invoke("std:index:join", {
    separator = ","
    input = [for s in names : invoke("std:index:upper", {
      input = s
    }).result]
  }).result
}

output "labelEntries" {
  value = invoke("std:index:join", {
    separator = ","
    input = [for k, v in labels : "${k}=${invoke("std:index:upper", {
      input = v
    }).result}"]
  }).result
}

output "shortNames" {
  value = invoke("std:index:join", {
    separator = ","
    input     = [for s in names : s if s != "bob"]
  }).result
}
