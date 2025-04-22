locals {
  a_list = ["a", "b", "c"]
  join_template_expr = <<EOT
%{for v in local.a_list~}
${v}
%{endfor~}
EOT

  tuple_cons_heredoc = [
<<EOT
oh baby give me
one more chance
to show you that I love you
EOT
  ]
}
