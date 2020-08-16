terraform {
  backend "s3" {
    key = "webprod1"
    bucket = "webtectrl-terra"
    region = "us-west-2"
  }
}

variable "table_name" {}
variable "tgbot_secret" {}

locals {
    wsauth_func_name = "wsauth_prod1"
}

data "aws_dynamodb_table" "main" {
    name = var.table_name
}

resource "aws_cloudwatch_log_group" "wsauth" {
    name = "/aws/lambda/${local.wsauth_func_name}"
    retention_in_days = 7
}

resource "aws_iam_role" "api" {
    name                = "apiprod1"
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

data "aws_iam_policy_document" "api" {
    statement {
        actions = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
        resources = ["*"]
    }
    statement {
        actions = [
                "dynamodb:BatchGet*",
                "dynamodb:DescribeStream",
                "dynamodb:DescribeTable",
                "dynamodb:Get*",
                "dynamodb:Query",
                "dynamodb:Scan",
                "dynamodb:BatchWrite*",
                "dynamodb:CreateTable",
                "dynamodb:Delete*",
                "dynamodb:Update*",
                "dynamodb:PutItem",
                "dynamodb:ListStreams"
            ]
        resources = [data.aws_dynamodb_table.main.arn]
    }
}

resource "aws_iam_policy" "api" {
    name = "api"
    policy = data.aws_iam_policy_document.api.json
}

resource "aws_iam_role_policy_attachment" "api" {
    role = aws_iam_role.api.name
    policy_arn = aws_iam_policy.api.arn
}

data "archive_file" "wsauth" {
    type = "zip"
    source_file = "${path.root}/../../lambda/wsauth/wsauth"
    output_path = "/tmp/wsauth.zip"
}

resource "aws_lambda_function" "wsauth" {
    function_name = local.wsauth_func_name
    runtime = "go1.x"
    handler = "wsauth"
    memory_size = 128
    timeout = 10
    role = aws_iam_role.api.arn
    filename = data.archive_file.wsauth.output_path
    source_code_hash = data.archive_file.wsauth.output_base64sha256
    environment  {
        variables = {
            TABLE_NAME = var.table_name
        }
    }
}

resource "aws_apigatewayv2_api" "wsapi" {
    name = "wsapi-prod1"
    protocol_type = "WEBSOCKET"
}

resource "aws_lambda_permission" "wsauth" {
    statement_id = "${aws_lambda_function.wsauth.function_name}Lambda"
    function_name = aws_lambda_function.wsauth.function_name
    action = "lambda:InvokeFunction"
    principal = "apigateway.amazonaws.com"
    source_arn = "${aws_apigatewayv2_api.wsapi.execution_arn}/*"
}

resource "aws_apigatewayv2_authorizer" "wsapi" {
    name = "wsapi-auth-prod1"
    api_id = aws_apigatewayv2_api.wsapi.id
    authorizer_type = "REQUEST"
    authorizer_uri = aws_lambda_function.wsauth.invoke_arn
    identity_sources = ["route.request.querystring.secret"]
}

resource "aws_apigatewayv2_route" "wsconn" {
    api_id = aws_apigatewayv2_api.wsapi.id
    route_key = "$connect"
    authorization_type = "CUSTOM"
    authorizer_id = aws_apigatewayv2_authorizer.wsapi.id
}

resource "aws_apigatewayv2_deployment" "wsapi" {
  api_id = aws_apigatewayv2_api.wsapi.id
  description = "prod1"

  depends_on = [aws_apigatewayv2_route.wsconn]

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_apigatewayv2_stage" "wsapi" {
  api_id = aws_apigatewayv2_api.wsapi.id
  deployment_id = aws_apigatewayv2_deployment.wsapi.id
  name   = "prod1"
}

output "api-url" {
    value = "${aws_apigatewayv2_api.wsapi.api_endpoint}/${aws_apigatewayv2_stage.wsapi.name}"
}
