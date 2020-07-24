terraform {
  backend "s3" {
    key = "webhook"
  }
}


resource "aws_iam_role" "webhook" {
    name                = "webhook"
  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": {
    "Action": "sts:AssumeRole",
    "Principal": {
      "Service": "lambda.amazonaws.com"
    },
    "Effect": "Allow"
  }
}
POLICY
}


resource "aws_iam_role_policy_attachment" "basic_lambda_execution" {
    role = aws_iam_role.webhook.name
    policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"

}

resource "aws_lambda_function" "tg_webhook" {
    function_name     = "tg_webhook"
    s3_bucket         = var.deploy_bucket  
    s3_key            = "lambda/tg-webhook.v${var.tg_lambda_ver}.zip"
    handler           = var.lambda_binary_name
    role              = aws_iam_role.webhook.arn
    runtime           = "go1.x"
    timeout           = 10
    memory_size       = 128
}

resource "aws_api_gateway_rest_api" "webhook" {
    name = "webhook"
}


resource "aws_api_gateway_resource" "tg_webhook_base" {
  rest_api_id = aws_api_gateway_rest_api.webhook.id
  parent_id   = aws_api_gateway_rest_api.webhook.root_resource_id
  path_part   = "tg"
}

resource "aws_api_gateway_resource" "tg_webhook" {
  rest_api_id = aws_api_gateway_rest_api.webhook.id
  parent_id   = aws_api_gateway_resource.tg_webhook_base.id
  path_part   = "{proxy+}"
}

resource "aws_api_gateway_method" "tg_webhook" { 
    rest_api_id   = aws_api_gateway_rest_api.webhook.id
    resource_id   = aws_api_gateway_resource.tg_webhook.id
    http_method   = "POST"
    authorization = "NONE"
}


resource "aws_api_gateway_integration" "tg_webhook" {
    rest_api_id             = aws_api_gateway_rest_api.webhook.id
    resource_id             = aws_api_gateway_resource.tg_webhook.id
    http_method             = aws_api_gateway_method.tg_webhook.http_method
    integration_http_method = "POST"
    type                    = "AWS_PROXY"
    uri                     = aws_lambda_function.tg_webhook.invoke_arn
}


resource "aws_lambda_permission" "tg_webhook" {
    statement_id    = "AllowExecutionFromAPIGateway"
    action          = "lambda:InvokeFunction"
    function_name   = aws_lambda_function.tg_webhook.function_name
    principal       = "apigateway.amazonaws.com"
    source_arn      = "${aws_api_gateway_rest_api.webhook.execution_arn}/*/*/*"
}


resource "aws_api_gateway_deployment" "tg_webhook" {
    depends_on       = [aws_api_gateway_integration.tg_webhook]
    rest_api_id      = aws_api_gateway_rest_api.webhook.id
    stage_name       = "prod1"
}



output tg-webhook-url {
    value = "${aws_api_gateway_deployment.tg_webhook.invoke_url}${aws_api_gateway_resource.tg_webhook.path}"
}
