output "url" {
  description = "Base URL for API Gateway stage."

  value = aws_apigatewayv2_stage.lambda.invoke_url
}

output "arn" {
  value = aws_lambda_function.hello_world.arn
}
