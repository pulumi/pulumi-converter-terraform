aList            = ["a", "b", "c"]
joinTemplateExpr = <<EOT
%{for v in aList~}${v}\n%{endfor~}
EOT
