terraform {
  backend "s3" {
    key = "prod1"
    bucket = "webtectrl-terra"
    region = "us-west-2"
  }
}

variable "table_name" {}
variable "tgbot_secret" {}

locals {
    func_name = "tgwebhook_prod1"
}

provider "telegram" {
    bot_token = var.tgbot_secret
}

resource "aws_sqs_queue" "errors" {
    name = "errors.fifo"
    fifo_queue = true
}

resource "aws_sqs_queue" "tgwebhook" {
    name = "tgwebhook.fifo"
    fifo_queue = true
    content_based_deduplication = true
    redrive_policy = jsonencode({
        deadLetterTargetArn = aws_sqs_queue.errors.arn,
        maxReceiveCount = 6
    })
}

resource "aws_cloudwatch_log_group" "tgwebhook" {
    name = "/aws/lambda/${local.func_name}"
    retention_in_days = 7
}

resource "aws_iam_role" "tgwebhook" {
    name                = "tgwebhook"
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

data "aws_iam_policy_document" "tgwebhook" {
    statement {
        actions = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
        resources = ["*"]
//        resources  = [aws_cloudwatch_log_group.tgwebhook.arn]
    }
    statement {
        actions = ["sqs:SendMessage"]
        resources = [aws_sqs_queue.tgwebhook.arn]
    }
}

resource "aws_iam_policy" "tgwebhook" {
    name = "tgwebhook"
    policy = data.aws_iam_policy_document.tgwebhook.json
}

resource "aws_iam_role_policy_attachment" "tgwebhook" {
    role = aws_iam_role.tgwebhook.name
    policy_arn = aws_iam_policy.tgwebhook.arn
}

data "archive_file" "tgwebhook" {
    type = "zip"
    source_file = "${path.root}/../../lambda/tgwebhook/tgwebhook"
    output_path = "/tmp/tgwebhook.zip"
}

resource "aws_lambda_function" "tgwebhook" {
    function_name = local.func_name
    runtime = "go1.x"
    handler = "tgwebhook"
    memory_size = 128
    timeout = 10
    role = aws_iam_role.tgwebhook.arn
    filename = data.archive_file.tgwebhook.output_path
    source_code_hash = data.archive_file.tgwebhook.output_base64sha256
    environment  {
        variables = {
            TGBOT_SECRET = var.tgbot_secret
        }
    }
}

resource "aws_apigatewayv2_api" "tgwebhook" {
    name = "prod1_tgwebhook"
    protocol_type = "HTTP"
    target = aws_lambda_function.tgwebhook.arn 
    route_key = "POST /"
}

resource "aws_lambda_permission" "tgwebhook" {
    statement_id = "tgwebhookLambda"
    function_name = aws_lambda_function.tgwebhook.function_name
    action = "lambda:InvokeFunction"
    principal = "apigateway.amazonaws.com"
    source_arn = "${aws_apigatewayv2_api.tgwebhook.execution_arn}/*/*/*"
}

resource "telegram_bot_webhook" "tgwebhook" {
    url  = aws_apigatewayv2_api.tgwebhook.api_endpoint
    max_connections = 100
}

output "webhook_url" {
    value = aws_apigatewayv2_api.tgwebhook.api_endpoint
}

