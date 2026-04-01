aList            = ["a", "b", "c"]
joinTemplateExpr = <<EOT
%{for v in aList~}${v}\n%{endfor~}
EOT

tupleConsHeredoc = [<<EOT
oh baby give me
one more chance
to show you that I love you
EOT
]
