
# A load of the examples in the docs use `path.module` which _should_ resolve to the file system path of #
# the current module, but tf2pulumi doesn't support that so we replace it with local.path_module.
pathModule = "some/path"

# Instead of using path.root use a local to not conflate testing of
# path.root conversion with uses of a path.
pathRoot = "root/path"

# The `can` examples make use of a local `foo`.
foo = {
  "bar" = "baz"
}

# The `nonsensitive` examples make use of a local `mixed_content`.
# We don't use jsondecode(var.mixed_content_json) here because we don't want to depend on the jsondecode function working.
mixedContent = {
  "password" = "hunter2"
}


# The `nonsensitive` examples make use of a variable `mixed_content_json`.
config "mixedContentJson" "string" {
}


# The `format` examples make use of a variable `name`.
config "name" "string" {
}


# The `matchkeys` example makes use of a resource with `count`.
resource "aResourceWithCount" "simple:index:resource" {
  __logicalName = "a_resource_with_count"
  options {
    range = 4
  }
  inputOne = "Hello ${range.value}"
  inputTwo = true
}


# Examples for abs
output "funcAbs0" {
  value = invoke("std:index:abs", {
    input = 23
  }).result
}
output "funcAbs1" {
  value = invoke("std:index:abs", {
    input = 0
  }).result
}
output "funcAbs2" {
  value = invoke("std:index:abs", {
    input = -12.4
  }).result
}



# Examples for abspath
output "funcAbspath" {
  value = invoke("std:index:abspath", {
    input = pathRoot
  }).result
}



# Examples for alltrue
output "funcAlltrue0" {
  value = invoke("std:index:alltrue", {
    input = ["true", true]
  }).result
}
output "funcAlltrue1" {
  value = invoke("std:index:alltrue", {
    input = [true, false]
  }).result
}



# Examples for anytrue
output "funcAnytrue0" {
  value = invoke("std:index:anytrue", {
    input = ["true"]
  }).result
}
output "funcAnytrue1" {
  value = invoke("std:index:anytrue", {
    input = [true]
  }).result
}
output "funcAnytrue2" {
  value = invoke("std:index:anytrue", {
    input = [true, false]
  }).result
}
output "funcAnytrue3" {
  value = invoke("std:index:anytrue", {
    input = []
  }).result
}



# Examples for base64decode
output "funcBase64decode" {
  value = invoke("std:index:base64decode", {
    input = "SGVsbG8gV29ybGQ="
  }).result
}



# Examples for base64encode
output "funcBase64encode" {
  value = invoke("std:index:base64encode", {
    input = "Hello World"
  }).result
}



# Examples for base64gzip
output "funcBase64gzip" {
  value = invoke("std:index:base64gzip", {
    input = "test"
  }).result
}



# Examples for base64sha256
output "funcBase64sha256" {
  value = invoke("std:index:base64sha256", {
    input = "hello world"
  }).result
}



# Examples for base64sha512
output "funcBase64sha512" {
  value = invoke("std:index:base64sha512", {
    input = "hello world"
  }).result
}



# Examples for basename
output "funcBasename" {
  value = invoke("std:index:basename", {
    input = "foo/bar/baz.txt"
  }).result
}



# Examples for bcrypt
output "funcBcrypt" {
  value = invoke("std:index:bcrypt", {
    input = "hello world"
  }).result
}



# Examples for can
output "funcCan0" {
  value = foo
}
output "funcCan1" {
  value = notImplemented("can(local.foo.bar)")
}
output "funcCan2" {
  value = notImplemented("can(local.foo.boop)")
}
output "funcCan3" {
  value = notImplemented("can(local.nonexist)")
}



# Examples for ceil
output "funcCeil0" {
  value = invoke("std:index:ceil", {
    input = 5
  }).result
}
output "funcCeil1" {
  value = invoke("std:index:ceil", {
    input = 5.1
  }).result
}



# Examples for chomp
output "funcChomp0" {
  value = invoke("std:index:chomp", {
    input = "hello\n"
  }).result
}
output "funcChomp1" {
  value = invoke("std:index:chomp", {
    input = "hello\r\n"
  }).result
}
output "funcChomp2" {
  value = invoke("std:index:chomp", {
    input = "hello\n\n"
  }).result
}



# Examples for chunklist
output "funcChunklist0" {
  value = invoke("std:index:chunklist", {
    input = ["a", "b", "c", "d", "e"]
    size  = 2
  }).result
}
output "funcChunklist1" {
  value = invoke("std:index:chunklist", {
    input = ["a", "b", "c", "d", "e"]
    size  = 1
  }).result
}



# Examples for cidrhost
output "funcCidrhost0" {
  value = invoke("std:index:cidrhost", {
    input = "10.12.112.0/20"
    host  = 16
  }).result
}
output "funcCidrhost1" {
  value = invoke("std:index:cidrhost", {
    input = "10.12.112.0/20"
    host  = 268
  }).result
}
output "funcCidrhost2" {
  value = invoke("std:index:cidrhost", {
    input = "fd00:fd12:3456:7890:00a2::/72"
    host  = 34
  }).result
}



# Examples for cidrnetmask
output "funcCidrnetmask" {
  value = invoke("std:index:cidrnetmask", {
    input = "172.16.0.0/12"
  }).result
}



# Examples for cidrsubnet
output "funcCidrsubnet0" {
  value = invoke("std:index:cidrsubnet", {
    input   = "172.16.0.0/12"
    newbits = 4
    netnum  = 2
  }).result
}
output "funcCidrsubnet1" {
  value = invoke("std:index:cidrsubnet", {
    input   = "10.1.2.0/24"
    newbits = 4
    netnum  = 15
  }).result
}
output "funcCidrsubnet2" {
  value = invoke("std:index:cidrsubnet", {
    input   = "fd00:fd12:3456:7890::/56"
    newbits = 16
    netnum  = 162
  }).result
}
output "funcCidrsubnet3" {
  value = invoke("std:index:cidrhost", {
    input = "10.1.2.240/28"
    host  = 1
  }).result
}
output "funcCidrsubnet4" {
  value = invoke("std:index:cidrhost", {
    input = "10.1.2.240/28"
    host  = 14
  }).result
}



# Examples for cidrsubnets
output "funcCidrsubnets0" {
  value = invoke("std:index:cidrsubnets", {
    input   = "10.1.0.0/16"
    newbits = [4, 4, 8, 4]
  }).result
}
output "funcCidrsubnets1" {
  value = invoke("std:index:cidrsubnets", {
    input   = "fd00:fd12:3456:7890::/56"
    newbits = [16, 16, 16, 32]
  }).result
}
output "funcCidrsubnets2" {
  value = [for cidrBlock in invoke("std:index:cidrsubnets", {
    input   = "10.0.0.0/8"
    newbits = [8, 8, 8, 8]
    }).result : invoke("std:index:cidrsubnets", {
    input   = cidrBlock
    newbits = [4, 4]
  }).result]
}



# Examples for coalesce
output "funcCoalesce0" {
  value = invoke("std:index:coalesce", {
    input = ["a", "b"]
  }).result
}
output "funcCoalesce1" {
  value = invoke("std:index:coalesce", {
    input = ["", "b"]
  }).result
}
output "funcCoalesce2" {
  value = invoke("std:index:coalesce", {
    input = [1, 2]
  }).result
}
output "funcCoalesce3" {
  value = invoke("std:index:coalesce", {
    input = ["", "b"]
  }).result
}
output "funcCoalesce4" {
  value = invoke("std:index:coalesce", {
    input = [1, "hello"]
  }).result
}
output "funcCoalesce5" {
  value = invoke("std:index:coalesce", {
    input = [true, "hello"]
  }).result
}
output "funcCoalesce6" {
  value = invoke("std:index:coalesce", {
    input = [{}, "hello"]
  }).result
}



# Examples for coalescelist
output "funcCoalescelist0" {
  value = invoke("std:index:coalescelist", {
    input = [["a", "b"], ["c", "d"]]
  }).result
}
output "funcCoalescelist1" {
  value = invoke("std:index:coalescelist", {
    input = [[], ["c", "d"]]
  }).result
}
output "funcCoalescelist2" {
  value = invoke("std:index:coalescelist", {
    input = [[], ["c", "d"]]
  }).result
}



# Examples for compact
output "funcCompact" {
  value = invoke("std:index:compact", {
    input = ["a", "", "b", null, "c"]
  }).result
}



# Examples for concat
output "funcConcat" {
  value = invoke("std:index:concat", {
    input = [["a", ""], ["b", "c"]]
  }).result
}



# Examples for contains
output "funcContains0" {
  value = invoke("std:index:contains", {
    input   = ["a", "b", "c"]
    element = "a"
  }).result
}
output "funcContains1" {
  value = invoke("std:index:contains", {
    input   = ["a", "b", "c"]
    element = "d"
  }).result
}



# Examples for csvdecode
output "funcCsvdecode" {
  value = invoke("std:index:csvdecode", {
    input = "a,b,c\n1,2,3\n4,5,6"
  }).result
}



# Examples for dirname
output "funcDirname" {
  value = invoke("std:index:dirname", {
    input = "foo/bar/baz.txt"
  }).result
}



# Examples for distinct
output "funcDistinct" {
  value = invoke("std:index:distinct", {
    input = ["a", "b", "a", "c", "d", "b"]
  }).result
}



# Examples for element
output "funcElement0" {
  value = element(["a", "b", "c"], 1)
}
output "funcElement1" {
  value = element(["a", "b", "c"], 3)
}
output "funcElement2" {
  value = element(["a", "b", "c"], -1)
}



# Examples for endswith
output "funcEndswith0" {
  value = invoke("std:index:endswith", {
    input  = "hello world"
    suffix = "world"
  }).result
}
output "funcEndswith1" {
  value = invoke("std:index:endswith", {
    input  = "hello world"
    suffix = "hello"
  }).result
}



# Examples for ephemeralasnull
output "funcEphemeralasnull" {
  value = notImplemented("ephemeralasnull(locals.foo)")
}



# Examples for file
output "funcFile" {
  value = invoke("std:index:file", {
    input = "${pathModule}/hello.txt"
  }).result
}



# Examples for filebase64
output "funcFilebase64" {
  value = invoke("std:index:filebase64", {
    input = "${pathModule}/hello.txt"
  }).result
}



# Examples for filebase64sha256
output "funcFilebase64sha256" {
  value = invoke("std:index:filebase64sha256", {
    input = "hello.txt"
  }).result
}



# Examples for filebase64sha512
output "funcFilebase64sha512" {
  value = invoke("std:index:filebase64sha512", {
    input = "hello.txt"
  }).result
}



# Examples for fileexists
output "funcFileexists" {
  value = invoke("std:index:fileexists", {
    input = "${pathModule}/hello.txt"
  }).result
}



# Examples for filemd5
output "funcFilemd5" {
  value = invoke("std:index:filemd5", {
    input = "hello.txt"
  }).result
}



# Examples for fileset
output "funcFileset0" {
  value = notImplemented("fileset(local.path_module,\"files/*.txt\")")
}
output "funcFileset1" {
  value = notImplemented("fileset(local.path_module,\"files/{hello,world}.txt\")")
}
output "funcFileset2" {
  value = notImplemented("fileset(\"$${local.path_module}/files\",\"*\")")
}
output "funcFileset3" {
  value = notImplemented("fileset(\"$${local.path_module}/files\",\"**\")")
}



# Examples for filesha1
output "funcFilesha1" {
  value = invoke("std:index:filesha1", {
    input = "hello.txt"
  }).result
}



# Examples for filesha256
output "funcFilesha256" {
  value = invoke("std:index:filesha256", {
    input = "hello.txt"
  }).result
}



# Examples for filesha512
output "funcFilesha512" {
  value = invoke("std:index:filesha512", {
    input = "hello.txt"
  }).result
}



# Examples for flatten
output "funcFlatten0" {
  value = invoke("std:index:flatten", {
    input = [["a", "b"], [], ["c"]]
  }).result
}
output "funcFlatten1" {
  value = invoke("std:index:flatten", {
    input = [[["a", "b"], []], ["c"]]
  }).result
}



# Examples for floor
output "funcFloor0" {
  value = invoke("std:index:floor", {
    input = 5
  }).result
}
output "funcFloor1" {
  value = invoke("std:index:floor", {
    input = 4.9
  }).result
}



# Examples for format
output "funcFormat0" {
  value = invoke("std:index:format", {
    input = "Hello, %s!"
    args  = ["Ander"]
  }).result
}
output "funcFormat1" {
  value = invoke("std:index:format", {
    input = "There are %d lights"
    args  = [4]
  }).result
}
output "funcFormat2" {
  value = invoke("std:index:format", {
    input = "Hello, %s!"
    args  = [name]
  }).result
}
output "funcFormat3" {
  value = "Hello, ${name}!"
}
output "funcFormat4" {
  value = invoke("std:index:format", {
    input = "%[1]s%[2]s%[1]s%[3]s"
    args  = ["/", "path", "file.tf"]
  }).result
}
output "funcFormat5" {
  value = invoke("std:index:format", {
    input = "%#v"
    args  = ["hello"]
  }).result
}
output "funcFormat6" {
  value = invoke("std:index:format", {
    input = "%#v"
    args  = [true]
  }).result
}
output "funcFormat7" {
  value = invoke("std:index:format", {
    input = "%#v"
    args  = [1]
  }).result
}
output "funcFormat8" {
  value = invoke("std:index:format", {
    input = "%#v"
    args = [{
      a = 1
    }]
  }).result
}
output "funcFormat9" {
  value = invoke("std:index:format", {
    input = "%#v"
    args  = [[true]]
  }).result
}
output "funcFormat10" {
  value = invoke("std:index:format", {
    input = "%#v"
    args  = [null]
  }).result
}



# Examples for formatdate
output "funcFormatdate0" {
  value = notImplemented("formatdate(\"DD MMM YYYY hh:mm ZZZ\",\"2018-01-02T23:12:01Z\")")
}
output "funcFormatdate1" {
  value = notImplemented("formatdate(\"EEEE, DD-MMM-YY hh:mm:ss ZZZ\",\"2018-01-02T23:12:01Z\")")
}
output "funcFormatdate2" {
  value = notImplemented("formatdate(\"EEE, DD MMM YYYY hh:mm:ss ZZZ\",\"2018-01-02T23:12:01-08:00\")")
}
output "funcFormatdate3" {
  value = notImplemented("formatdate(\"MMM DD, YYYY\",\"2018-01-02T23:12:01Z\")")
}
output "funcFormatdate4" {
  value = notImplemented("formatdate(\"HH:mmaa\",\"2018-01-02T23:12:01Z\")")
}
output "funcFormatdate5" {
  value = notImplemented("formatdate(\"h'h'mm\",\"2018-01-02T23:12:01-08:00\")")
}
output "funcFormatdate6" {
  value = notImplemented("formatdate(\"H 'o''clock'\",\"2018-01-02T23:12:01-08:00\")")
}



# Examples for formatlist
output "funcFormatlist0" {
  value = invoke("std:index:formatlist", {
    input = "Hello, %s!"
    args  = [["Valentina", "Ander", "Olivia", "Sam"]]
  }).result
}
output "funcFormatlist1" {
  value = invoke("std:index:formatlist", {
    input = "%s, %s!"
    args  = ["Salutations", ["Valentina", "Ander", "Olivia", "Sam"]]
  }).result
}



# Examples for indent
output "funcIndent" {
  value = "  items: ${invoke("std:index:indent", {
    spaces = 2
    input  = "[\n  foo,\n  bar,\n]\n"
  }).result}"
}



# Examples for index
output "funcIndex" {
  value = notImplemented("index([\"a\",\"b\",\"c\"],\"b\")")
}



# Examples for issensitive
output "funcIssensitive0" {
  value = notImplemented("issensitive(sensitive(\"secret\"))")
}
output "funcIssensitive1" {
  value = notImplemented("issensitive(\"hello\")")
}
output "funcIssensitive2" {
  value = notImplemented("issensitive(var.my-var-with-sensitive-set-to-true)")
}



# Examples for join
output "funcJoin0" {
  value = invoke("std:index:join", {
    separator = "-"
    input     = ["foo", "bar", "baz"]
  }).result
}
output "funcJoin1" {
  value = invoke("std:index:join", {
    separator = ", "
    input     = ["foo", "bar", "baz"]
  }).result
}
output "funcJoin2" {
  value = invoke("std:index:join", {
    separator = ", "
    input     = ["foo"]
  }).result
}



# Examples for jsondecode
output "funcJsondecode0" {
  value = invoke("std:index:jsondecode", {
    input = "{\"hello\": \"world\"}"
  }).result
}
output "funcJsondecode1" {
  value = invoke("std:index:jsondecode", {
    input = "true"
  }).result
}



# Examples for jsonencode
output "funcJsonencode" {
  value = toJSON({
    "hello" = "world"
  })
}



# Examples for keys
output "funcKeys" {
  value = invoke("std:index:keys", {
    input = {
      a = 1
      c = 2
      d = 3
    }
  }).result
}



# Examples for length
output "funcLength0" {
  value = length([])
}
output "funcLength1" {
  value = length(["a", "b"])
}
output "funcLength2" {
  value = length({
    "a" = "b"
  })
}
output "funcLength3" {
  value = length("hello")
}
output "funcLength4" {
  value = length("👾🕹️")
}



# Examples for list
output "funcList" {
  value = [1, 2, 3]
}



# Examples for log
output "funcLog0" {
  value = invoke("std:index:log", {
    base  = 50
    input = 10
  }).result
}
output "funcLog1" {
  value = invoke("std:index:log", {
    base  = 16
    input = 2
  }).result
}
output "funcLog2" {
  value = invoke("std:index:ceil", {
    input = invoke("std:index:log", {
      base  = 15
      input = 2
    }).result
  }).result
}
output "funcLog3" {
  value = invoke("std:index:ceil", {
    input = invoke("std:index:log", {
      base  = 16
      input = 2
    }).result
  }).result
}
output "funcLog4" {
  value = invoke("std:index:ceil", {
    input = invoke("std:index:log", {
      base  = 17
      input = 2
    }).result
  }).result
}



# Examples for lookup
output "funcLookup0" {
  value = invoke("std:index:lookup", {
    map = {
      a = "ay"
      b = "bee"
    }
    key     = "a"
    default = "what?"
  }).result
}
output "funcLookup1" {
  value = invoke("std:index:lookup", {
    map = {
      a = "ay"
      b = "bee"
    }
    key     = "c"
    default = "what?"
  }).result
}



# Examples for lower
output "funcLower0" {
  value = invoke("std:index:lower", {
    input = "HELLO"
  }).result
}
output "funcLower1" {
  value = invoke("std:index:lower", {
    input = "АЛЛО!"
  }).result
}



# Examples for map
output "funcMap" {
  value = notImplemented("map(\"a\",\"b\",\"c\",\"d\")")
}



# Examples for matchkeys
output "funcMatchkeys0" {
  value = notImplemented("matchkeys([\"i-123\",\"i-abc\",\"i-def\"],[\"us-west\",\"us-east\",\"us-east\"],[\"us-east\"])")
}
output "funcMatchkeys1" {
  value = [for i, z in {
    "i-123" = "us-west"
    "i-abc" = "us-east"
    "i-def" = "us-east"
  } : i if z == "us-east"]
}
output "funcMatchkeys2" {
  value = [for x in [{
    id   = "i-123"
    zone = "us-west"
    }, {
    id   = "i-abc"
    zone = "us-east"
  }] : x.id if x.zone == "us-east"]
}
output "funcMatchkeys3" {
  value = [for x in aResourceWithCount : x.id if x.inputOne == "us-east-1a"]
}



# Examples for max
output "funcMax0" {
  value = invoke("std:index:max", {
    input = [12, 54, 3]
  }).result
}
output "funcMax1" {
  value = invoke("std:index:max", {
    input = [12, 54, 3]
  }).result
}



# Examples for md5
output "funcMd5" {
  value = invoke("std:index:md5", {
    input = "hello world"
  }).result
}



# Examples for merge
output "funcMerge0" {
  value = invoke("std:index:merge", {
    input = [{
      a = "b"
      c = "d"
      }, {
      e = "f"
      c = "z"
    }]
  }).result
}
output "funcMerge1" {
  value = invoke("std:index:merge", {
    input = [{
      a = "b"
      }, {
      a = [1, 2]
      c = "z"
      }, {
      d = 3
    }]
  }).result
}
output "funcMerge2" {
  value = invoke("std:index:merge", {
    input = [{
      a = "b"
      c = "d"
      }, {}, {
      e = "f"
      c = "z"
    }]
  }).result
}



# Examples for min
output "funcMin0" {
  value = invoke("std:index:min", {
    input = [12, 54, 3]
  }).result
}
output "funcMin1" {
  value = invoke("std:index:min", {
    input = [12, 54, 3]
  }).result
}



# Examples for nonsensitive
output "funcNonsensitive0" {
  value = mixedContentJson
}
output "funcNonsensitive1" {
  value = mixedContent
}
output "funcNonsensitive2" {
  value = mixedContent["password"]
}
output "funcNonsensitive3" {
  value = notImplemented("nonsensitive(local.mixed_content[\"username\"])")
}
output "funcNonsensitive4" {
  value = notImplemented("nonsensitive(\"clear\")")
}
output "funcNonsensitive5" {
  value = notImplemented("nonsensitive(var.mixed_content_json)")
}
output "funcNonsensitive6" {
  value = notImplemented("nonsensitive(local.mixed_content)")
}
output "funcNonsensitive7" {
  value = notImplemented("nonsensitive(local.mixed_content[\"password\"])")
}



# Examples for one
output "funcOne0" {
  value = notImplemented("one([])")
}
output "funcOne1" {
  value = notImplemented("one([\"hello\"])")
}
output "funcOne2" {
  value = notImplemented("one([\"hello\",\"goodbye\"])")
}
output "funcOne3" {
  value = notImplemented("one(toset([]))")
}
output "funcOne4" {
  value = notImplemented("one(toset([\"hello\"]))")
}
output "funcOne5" {
  value = notImplemented("one(toset([\"hello\",\"goodbye\"]))")
}



# Examples for parseint
output "funcParseint0" {
  value = invoke("std:index:parseint", {
    input = "100"
    base  = 10
  }).result
}
output "funcParseint1" {
  value = invoke("std:index:parseint", {
    input = "FF"
    base  = 16
  }).result
}
output "funcParseint2" {
  value = invoke("std:index:parseint", {
    input = "-10"
    base  = 16
  }).result
}
output "funcParseint3" {
  value = invoke("std:index:parseint", {
    input = "1011111011101111"
    base  = 2
  }).result
}
output "funcParseint4" {
  value = invoke("std:index:parseint", {
    input = "aA"
    base  = 62
  }).result
}
output "funcParseint5" {
  value = invoke("std:index:parseint", {
    input = "12"
    base  = 2
  }).result
}



# Examples for pathexpand
output "funcPathexpand0" {
  value = invoke("std:index:pathexpand", {
    input = "~/.ssh/id_rsa"
  }).result
}
output "funcPathexpand1" {
  value = invoke("std:index:pathexpand", {
    input = "/etc/resolv.conf"
  }).result
}



# Examples for plantimestamp
output "funcPlantimestamp" {
  value = notImplemented("plantimestamp()")
}



# Examples for pow
output "funcPow0" {
  value = invoke("std:index:pow", {
    base     = 3
    exponent = 2
  }).result
}
output "funcPow1" {
  value = invoke("std:index:pow", {
    base     = 4
    exponent = 0
  }).result
}



# Examples for range
output "funcRange0" {
  value = invoke("std:index:range", {
    limit = 3
  }).result
}
output "funcRange1" {
  value = invoke("std:index:range", {
    limit = 1
    start = 4
  }).result
}
output "funcRange2" {
  value = invoke("std:index:range", {
    limit = 1
    start = 8
    step  = 2
  }).result
}
output "funcRange3" {
  value = invoke("std:index:range", {
    limit = 1
    start = 4
    step  = 0.5
  }).result
}
output "funcRange4" {
  value = invoke("std:index:range", {
    limit = 4
    start = 1
  }).result
}
output "funcRange5" {
  value = invoke("std:index:range", {
    limit = 10
    start = 5
    step  = -2
  }).result
}



# Examples for regex
output "funcRegex0" {
  value = invoke("std:index:regex", {
    pattern = "[a-z]+"
    string  = "53453453.345345aaabbbccc23454"
  }).result
}
output "funcRegex1" {
  value = invoke("std:index:regex", {
    pattern = "(\\d\\d\\d\\d)-(\\d\\d)-(\\d\\d)"
    string  = "2019-02-01"
  }).result
}
output "funcRegex2" {
  value = invoke("std:index:regex", {
    pattern = "^(?:(?P<scheme>[^:/?#]+):)?(?://(?P<authority>[^/?#]*))?"
    string  = "https://terraform.io/docs/"
  }).result
}
output "funcRegex3" {
  value = invoke("std:index:regex", {
    pattern = "[a-z]+"
    string  = "53453453.34534523454"
  }).result
}



# Examples for regexall
output "funcRegexall0" {
  value = invoke("std:index:regexall", {
    pattern = "[a-z]+"
    string  = "1234abcd5678efgh9"
  }).result
}
output "funcRegexall1" {
  value = length(invoke("std:index:regexall", {
    pattern = "[a-z]+"
    string  = "1234abcd5678efgh9"
  }).result)
}
output "funcRegexall2" {
  value = length(invoke("std:index:regexall", {
    pattern = "[a-z]+"
    string  = "123456789"
  }).result) > 0
}



# Examples for replace
output "funcReplace0" {
  value = invoke("std:index:replace", {
    text    = "1 + 2 + 3"
    search  = "+"
    replace = "-"
  }).result
}
output "funcReplace1" {
  value = invoke("std:index:replace", {
    text    = "hello world"
    search  = "/w.*d/"
    replace = "everybody"
  }).result
}



# Examples for reverse
output "funcReverse" {
  value = notImplemented("reverse([1,2,3])")
}



# Examples for rsadecrypt
output "funcRsadecrypt" {
  value = invoke("std:index:rsadecrypt", {
    cipherText = invoke("std:index:filebase64", {
      input = "${pathModule}/ciphertext"
    }).result
    key = invoke("std:index:file", {
      input = "privatekey.pem"
    }).result
  }).result
}



# Examples for sensitive
output "funcSensitive0" {
  value = secret(1)
}
output "funcSensitive1" {
  value = secret("hello")
}
output "funcSensitive2" {
  value = secret([])
}



# Examples for setintersection
output "funcSetintersection0" {
  value = invoke("std:index:setintersection", {
    input = [["a", "b"], ["b", "c"], ["b", "d"]]
  }).result
}
output "funcSetintersection1" {
  value = invoke("std:index:setintersection", {
    input = [[3, 3.3, 4], [4, 3.3, 65, 99], [4, 3.3]]
  }).result
}
output "funcSetintersection2" {
  value = invoke("std:index:setintersection", {
    input = [["bob", "jane", 3], ["jane", 3, "ajax", 10], ["3", "jane", 26, "nomad"]]
  }).result
}



# Examples for setproduct
output "funcSetproduct0" {
  value = notImplemented("setproduct([\"development\",\"staging\",\"production\"],[])")
}
output "funcSetproduct1" {
  value = notImplemented("setproduct([\"a\"],[\"b\"])")
}
output "funcSetproduct2" {
  value = notImplemented("setproduct([\"staging\",\"production\"],[\"a\",2])")
}



# Examples for setsubtract
output "funcSetsubtract0" {
  value = notImplemented("setsubtract([\"a\",\"b\",\"c\"],[\"a\",\"c\"])")
}
output "funcSetsubtract1" {
  value = notImplemented("setunion(setsubtract([\"a\",\"b\",\"c\"],[\"a\",\"c\",\"d\"]),setsubtract([\"a\",\"c\",\"d\"],[\"a\",\"b\",\"c\"]))")
}



# Examples for setunion
output "funcSetunion" {
  value = notImplemented("setunion([\"a\",\"b\"],[\"b\",\"c\"],[\"d\"])")
}



# Examples for sha1
output "funcSha1" {
  value = invoke("std:index:sha1", {
    input = "hello world"
  }).result
}



# Examples for sha256
output "funcSha256" {
  value = invoke("std:index:sha256", {
    input = "hello world"
  }).result
}



# Examples for sha512
output "funcSha512" {
  value = invoke("std:index:sha512", {
    input = "hello world"
  }).result
}



# Examples for signum
output "funcSignum0" {
  value = invoke("std:index:signum", {
    input = -13
  }).result
}
output "funcSignum1" {
  value = invoke("std:index:signum", {
    input = 0
  }).result
}
output "funcSignum2" {
  value = invoke("std:index:signum", {
    input = 344
  }).result
}



# Examples for slice
output "funcSlice" {
  value = invoke("std:index:slice", {
    list = ["a", "b", "c", "d"]
    from = 1
    to   = 3
  }).result
}



# Examples for sort
output "funcSort" {
  value = invoke("std:index:sort", {
    input = ["e", "d", "a", "x"]
  }).result
}



# Examples for split
output "funcSplit0" {
  value = invoke("std:index:split", {
    separator = ","
    text      = "foo,bar,baz"
  }).result
}
output "funcSplit1" {
  value = invoke("std:index:split", {
    separator = ","
    text      = "foo"
  }).result
}
output "funcSplit2" {
  value = invoke("std:index:split", {
    separator = ","
    text      = ""
  }).result
}



# Examples for startswith
output "funcStartswith0" {
  value = invoke("std:index:startswith", {
    input  = "hello world"
    prefix = "hello"
  }).result
}
output "funcStartswith1" {
  value = invoke("std:index:startswith", {
    input  = "hello world"
    prefix = "world"
  }).result
}



# Examples for strcontains
output "funcStrcontains0" {
  value = notImplemented("strcontains(\"hello world\",\"wor\")")
}
output "funcStrcontains1" {
  value = notImplemented("strcontains(\"hello world\",\"wod\")")
}



# Examples for strrev
output "funcStrrev0" {
  value = invoke("std:index:strrev", {
    input = "hello"
  }).result
}
output "funcStrrev1" {
  value = invoke("std:index:strrev", {
    input = "a ☃"
  }).result
}



# Examples for substr
output "funcSubstr0" {
  value = invoke("std:index:substr", {
    input  = "hello world"
    offset = 1
    length = 4
  }).result
}
output "funcSubstr1" {
  value = invoke("std:index:substr", {
    input  = "🤔🤷"
    offset = 0
    length = 1
  }).result
}
output "funcSubstr2" {
  value = invoke("std:index:substr", {
    input  = "hello world"
    offset = -5
    length = -1
  }).result
}
output "funcSubstr3" {
  value = invoke("std:index:substr", {
    input  = "hello world"
    offset = 6
    length = 10
  }).result
}



# Examples for sum
output "funcSum" {
  value = invoke("std:index:sum", {
    input = [10, 13, 6, 4.5]
  }).result
}



# Examples for templatefile
output "funcTemplatefile0" {
  value = notImplemented("templatefile(\"$${path.module}/backends.tftpl\",{port=8080,ip_addrs=[\"10.0.0.1\",\"10.0.0.2\"]})")
}
output "funcTemplatefile1" {
  value = notImplemented("templatefile(\n\"$${path.module}/config.tftpl\",\n{\nconfig={\n\"x\"=\"y\"\n\"foo\"=\"bar\"\n\"key\"=\"value\"\n}\n}\n)")
}



# Examples for templatestring
output "funcTemplatestring" {
  value = notImplemented("templatestring(\"$${var.foo}\",{foo=\"bar\"})")
}



# Examples for terraform-applying
output "funcTerraform-applying" {
  value = notImplemented("terraform.applying")
}



# Examples for terraform-decode_tfvars
output "funcTerraform-decodeTfvars" {
  value = notImplemented("provider::terraform::decode_tfvars(\"example = \\\"Hello!\\\"\")")
}



# Examples for terraform-encode_expr
output "funcTerraform-encodeExpr" {
  value = notImplemented("provider::terraform::encode_expr(locals.foo)")
}



# Examples for terraform-encode_tfvars
output "funcTerraform-encodeTfvars" {
  value = notImplemented("provider::terraform::encode_tfvars({example=\"Hello!\"})")
}



# Examples for textdecodebase64
output "funcTextdecodebase64" {
  value = notImplemented("textdecodebase64(\"SABlAGwAbABvACAAVwBvAHIAbABkAA==\",\"UTF-16LE\")")
}



# Examples for textencodebase64
output "funcTextencodebase64" {
  value = notImplemented("textencodebase64(\"Hello World\",\"UTF-16LE\")")
}



# Examples for timeadd
output "funcTimeadd0" {
  value = invoke("std:index:timeadd", {
    duration  = "2024-08-16T12:45:05Z"
    timestamp = "10m"
  }).result
}
output "funcTimeadd1" {
  value = invoke("std:index:timeadd", {
    duration  = "2024-08-16T12:45:05Z"
    timestamp = "-10m"
  }).result
}



# Examples for timecmp
output "funcTimecmp0" {
  value = invoke("std:index:timecmp", {
    timestampa = "2017-11-22T00:00:00Z"
    timestampb = "2017-11-22T00:00:00Z"
  }).result
}
output "funcTimecmp1" {
  value = invoke("std:index:timecmp", {
    timestampa = "2017-11-22T00:00:00Z"
    timestampb = "2017-11-22T01:00:00Z"
  }).result
}
output "funcTimecmp2" {
  value = invoke("std:index:timecmp", {
    timestampa = "2017-11-22T01:00:00Z"
    timestampb = "2017-11-22T00:00:00Z"
  }).result
}
output "funcTimecmp3" {
  value = invoke("std:index:timecmp", {
    timestampa = "2017-11-22T01:00:00Z"
    timestampb = "2017-11-22T00:00:00-01:00"
  }).result
}



# Examples for timestamp
output "funcTimestamp" {
  value = invoke("std:index:timestamp", {}).result
}



# Examples for title
output "funcTitle" {
  value = invoke("std:index:title", {
    input = "hello world"
  }).result
}



# Examples for tobool
output "funcTobool0" {
  value = invoke("std:index:tobool", {
    input = true
  }).result
}
output "funcTobool1" {
  value = invoke("std:index:tobool", {
    input = "true"
  }).result
}
output "funcTobool2" {
  value = invoke("std:index:tobool", {
    input = null
  }).result
}
output "funcTobool3" {
  value = invoke("std:index:tobool", {
    input = "no"
  }).result
}
output "funcTobool4" {
  value = invoke("std:index:tobool", {
    input = 1
  }).result
}



# Examples for tolist
output "funcTolist0" {
  value = ["a", "b", "c"]
}
output "funcTolist1" {
  value = ["a", "b", 3]
}



# Examples for tomap
output "funcTomap0" {
  value = notImplemented("tomap({\"a\"=1,\"b\"=2})")
}
output "funcTomap1" {
  value = notImplemented("tomap({\"a\"=\"foo\",\"b\"=true})")
}



# Examples for tonumber
output "funcTonumber0" {
  value = notImplemented("tonumber(1)")
}
output "funcTonumber1" {
  value = notImplemented("tonumber(\"1\")")
}
output "funcTonumber2" {
  value = notImplemented("tonumber(null)")
}
output "funcTonumber3" {
  value = notImplemented("tonumber(\"no\")")
}



# Examples for toset
output "funcToset0" {
  value = invoke("std:index:toset", {
    input = ["a", "b", "c"]
  }).result
}
output "funcToset1" {
  value = invoke("std:index:toset", {
    input = ["a", "b", 3]
  }).result
}
output "funcToset2" {
  value = invoke("std:index:toset", {
    input = ["c", "b", "b"]
  }).result
}



# Examples for tostring
output "funcTostring0" {
  value = notImplemented("tostring(\"hello\")")
}
output "funcTostring1" {
  value = notImplemented("tostring(1)")
}
output "funcTostring2" {
  value = notImplemented("tostring(true)")
}
output "funcTostring3" {
  value = notImplemented("tostring(null)")
}
output "funcTostring4" {
  value = notImplemented("tostring([])")
}



# Examples for transpose
output "funcTranspose" {
  value = invoke("std:index:transpose", {
    input = {
      "a" = ["1", "2"]
      "b" = ["2", "3"]
    }
  }).result
}



# Examples for trim
output "funcTrim0" {
  value = invoke("std:index:trim", {
    input  = "?!hello?!"
    cutset = "!?"
  }).result
}
output "funcTrim1" {
  value = invoke("std:index:trim", {
    input  = "foobar"
    cutset = "far"
  }).result
}
output "funcTrim2" {
  value = invoke("std:index:trim", {
    input  = "   hello! world.!  "
    cutset = "! "
  }).result
}



# Examples for trimprefix
output "funcTrimprefix0" {
  value = invoke("std:index:trimprefix", {
    input  = "helloworld"
    prefix = "hello"
  }).result
}
output "funcTrimprefix1" {
  value = invoke("std:index:trimprefix", {
    input  = "helloworld"
    prefix = "cat"
  }).result
}
output "funcTrimprefix2" {
  value = invoke("std:index:trimprefix", {
    input  = "--hello"
    prefix = "-"
  }).result
}



# Examples for trimspace
output "funcTrimspace" {
  value = invoke("std:index:trimspace", {
    input = "  hello\n\n"
  }).result
}



# Examples for trimsuffix
output "funcTrimsuffix0" {
  value = invoke("std:index:trimsuffix", {
    input  = "helloworld"
    suffix = "world"
  }).result
}
output "funcTrimsuffix1" {
  value = invoke("std:index:trimsuffix", {
    input  = "helloworld"
    suffix = "cat"
  }).result
}
output "funcTrimsuffix2" {
  value = invoke("std:index:trimsuffix", {
    input  = "hello--"
    suffix = "-"
  }).result
}



# Examples for try
output "funcTry0" {
  value = foo
}
output "funcTry1" {
  value = notImplemented("try(local.foo.bar,\"fallback\")")
}
output "funcTry2" {
  value = notImplemented("try(local.foo.boop,\"fallback\")")
}
output "funcTry3" {
  value = notImplemented("try(local.nonexist,\"fallback\")")
}



# Examples for type
output "funcType0" {
  value = notImplemented("type(var.list)")
}
output "funcType1" {
  value = notImplemented("type(local.default_list)")
}



# Examples for upper
output "funcUpper0" {
  value = invoke("std:index:upper", {
    input = "hello"
  }).result
}
output "funcUpper1" {
  value = invoke("std:index:upper", {
    input = "алло!"
  }).result
}



# Examples for urlencode
output "funcUrlencode0" {
  value = invoke("std:index:urlencode", {
    input = "Hello World!"
  }).result
}
output "funcUrlencode1" {
  value = invoke("std:index:urlencode", {
    input = "☃"
  }).result
}
output "funcUrlencode2" {
  value = "http://example.com/search?q=${invoke("std:index:urlencode", {
    input = "terraform urlencode"
  }).result}"
}



# Examples for uuid
output "funcUuid" {
  value = invoke("std:index:uuid", {}).result
}



# Examples for uuidv5
output "funcUuidv50" {
  value = notImplemented("uuidv5(\"dns\",\"www.terraform.io\")")
}
output "funcUuidv51" {
  value = notImplemented("uuidv5(\"url\",\"https://www.terraform.io/\")")
}
output "funcUuidv52" {
  value = notImplemented("uuidv5(\"oid\",\"1.3.6.1.4\")")
}
output "funcUuidv53" {
  value = notImplemented("uuidv5(\"x500\",\"CN=Example,C=GB\")")
}
output "funcUuidv54" {
  value = notImplemented("uuidv5(\"6ba7b810-9dad-11d1-80b4-00c04fd430c8\",\"www.terraform.io\")")
}
output "funcUuidv55" {
  value = notImplemented("uuidv5(\"743ac3c0-3bf7-4a5b-9e6c-59360447c757\",\"LIBS:diskfont.library\")")
}



# Examples for values
output "funcValues" {
  value = notImplemented("values({a=3,c=2,d=1})")
}



# Examples for yamldecode
output "funcYamldecode0" {
  value = notImplemented("yamldecode(\"hello: world\")")
}
output "funcYamldecode1" {
  value = notImplemented("yamldecode(\"true\")")
}
output "funcYamldecode2" {
  value = notImplemented("yamldecode(\"{a: &foo [1, 2, 3], b: *foo}\")")
}
output "funcYamldecode3" {
  value = notImplemented("yamldecode(\"{a: &foo [1, *foo, 3]}\")")
}
output "funcYamldecode4" {
  value = notImplemented("yamldecode(\"{a: !not-supported foo}\")")
}



# Examples for yamlencode
output "funcYamlencode0" {
  value = notImplemented("yamlencode({\"a\":\"b\",\"c\":\"d\"})")
}
output "funcYamlencode1" {
  value = notImplemented("yamlencode({\"foo\":[1,2,3],\"bar\":\"baz\"})")
}
output "funcYamlencode2" {
  value = notImplemented("yamlencode({\"foo\":[1,{\"a\":\"b\",\"c\":\"d\"},3],\"bar\":\"baz\"})")
}



# Examples for zipmap
output "funcZipmap" {
  value = notImplemented("zipmap([\"a\",\"b\"],[1,2])")
}
