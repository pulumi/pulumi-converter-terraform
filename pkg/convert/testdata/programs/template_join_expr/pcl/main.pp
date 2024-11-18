aList            = ["a", "b", "c"]
joinTemplateExpr = "%{for v in aList~}${v}%{endfor~}"
ifTemplateExpr   = "${true ? "true" : "false"}"
