variable "deploy_bucket" {
    default = "webtectrl-deploy"
}

variable "viber_webhook_s3_key" {
    default = "lambda/viber-webhook.zip"
}

variable "lambda_binary_name" {
    default = "main"
}
