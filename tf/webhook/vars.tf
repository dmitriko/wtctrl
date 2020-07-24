variable "deploy_bucket" {
    default = "webtectrl-deploy"
}

variable "lambda_binary_name" {
    default = "main"
}

variable "tg_lambda_ver" {
    description = "Version of Telegram lambda to put in play"
}

