terraform {
  backend "s3" {
    key = "prod1"
    bucket = "webtectrl-terra"
    region = "us-west-2"
  }
}

variable "table_name" {}
variable "tgbot_secret" {}
variable "speech_key" {}
variable "azure_region" {}

locals {
    webhook_func_name = "tgwebhook_prod1"
    dstream_func_name = "dstream_prod1"
}

provider "telegram" {
    bot_token = var.tgbot_secret
}

resource "aws_s3_bucket" "images" {
    bucket = "wtctrl-udatab"
    acl = "private"
    
    lifecycle_rule {
        enabled = true
        expiration {
            days = 90
        }
    }
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
    name = "/aws/lambda/${local.webhook_func_name}"
    retention_in_days = 7
}

resource "aws_cloudwatch_log_group" "dstream" {
    name = "/aws/lambda/${local.dstream_func_name}"
    retention_in_days = 7
}

resource "aws_iam_role" "lambda" {
    name                = "lambda"
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

data "aws_iam_policy_document" "lambda" {
    statement {
        actions = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
        resources = ["*"]
    }
    statement {
        actions = ["execute-api:Invoke", "execute-api:ManageConnections"]
        resources = ["arn:aws:execute-api:*:*:*"]
    }
    statement {
        actions = ["s3:*"]
        resources = ["${aws_s3_bucket.images.arn}/*", aws_s3_bucket.images.arn]
    }
    statement {
        actions = ["sqs:SendMessage"]
        resources = [aws_sqs_queue.tgwebhook.arn]
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
        resources = [aws_dynamodb_table.main.arn, aws_dynamodb_table.main.stream_arn]
    }
}

resource "aws_iam_policy" "lambda" {
    name = "lambda"
    policy = data.aws_iam_policy_document.lambda.json
}

resource "aws_iam_role_policy_attachment" "lambda" {
    role = aws_iam_role.lambda.name
    policy_arn = aws_iam_policy.lambda.arn
}

data "archive_file" "tgwebhook" {
    type = "zip"
    source_file = "${path.root}/../../lambda/tgwebhook/tgwebhook"
    output_path = "/tmp/tgwebhook.zip"
}

resource "aws_lambda_function" "tgwebhook" {
    function_name = local.webhook_func_name
    runtime = "go1.x"
    handler = "tgwebhook"
    memory_size = 128
    timeout = 10
    role = aws_iam_role.lambda.arn
    filename = data.archive_file.tgwebhook.output_path
    source_code_hash = data.archive_file.tgwebhook.output_base64sha256
    environment  {
        variables = {
            TGBOT_SECRET = var.tgbot_secret
            TABLE_NAME = var.table_name
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

data "archive_file" "dstream" {
    type = "zip"
    source_file = "${path.root}/../../lambda/dstream/dstream"
    output_path = "/tmp/dstream.zip"
}

resource "aws_lambda_function" "dstream" {
    function_name = local.dstream_func_name
    runtime = "go1.x"
    handler = "dstream"
    memory_size = 128
    timeout = 10
    role = aws_iam_role.lambda.arn
    filename = data.archive_file.dstream.output_path
    source_code_hash = data.archive_file.dstream.output_base64sha256
    environment  {
        variables = {
            TGBOT_SECRET = var.tgbot_secret
            TABLE_NAME = var.table_name
            AZURE_SPEECH2TEXT_KEY = var.speech_key
            AZURE_REGION = var.azure_region
            IMG_BUCKET = aws_s3_bucket.images.id
        }
    }
}

resource "aws_lambda_event_source_mapping" "dstream" {
  event_source_arn  = aws_dynamodb_table.main.stream_arn
  function_name     = aws_lambda_function.dstream.arn
  starting_position = "LATEST"
}
