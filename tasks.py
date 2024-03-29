import io
import os
from pathlib import Path

from invoke import task

import boto3
from botocore.errorfactory import ClientError
import IPython


DEPLOY_BUCKET = "webtectrl-deploy"
try:
    REGION = os.environ['AWS_DEFAULT_REGION']
except KeyError:
    raise RuntimError("Please, source environment file")


def s3_backup(bucket, key_name, backup_name):
    "Copy s3 key to a new backup location if exists"
    s3 = boto3.resource('s3')
    try:
        s3.Object(bucket, key_name).load()
    except ClientError:
        return
    s3.Object(bucket, backup_name).copy_from(
                CopySource="{}/{}".format(bucket, key_name))


@task
def lambda_deploy(c, path, version, bucket=DEPLOY_BUCKET):
    "Build golang lambda, zip it and upload to deploy bucket"
    s3 = boto3.client('s3')
    bin_name = "main"  # name of binary file
    zip_name = bin_name + '.zip'
    path = Path(path)
    # it will be lambda/tg-webhook.v1.0.1.zip
    s3key = "{}/{}.v{}.zip".format(path.parts[-2], path.parts[-1], version)
    s3key_prev = "{}/{}.prev.zip".format(path.parts[-2], path.parts[-1])
    s3_backup(bucket, s3key, s3key_prev)
    with c.cd(str(path)):
        c.run('go get .')
        c.run('GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o {}'.format(
            bin_name))
        c.run("zip -j {} {}".format(zip_name, bin_name))
        zip_path = path / zip_name
        s3.upload_file(str(zip_path), bucket, s3key)
        c.run('rm -f {} {}'.format(bin_name, zip_name))


@task
def terra_init(c, path, bucket=None):
    "Init terraform in dir with state bucket and region"
    bucket = bucket or os.environ.get('TF_VAR_terrabucket')
    if not bucket:
        raise RuntimeError(
                "Bucket should be provided via TF_VAR_terrabucket env var")
    with c.cd(path):
        c.run('terraform init -backend-config bucket=' + bucket + \
        ' -backend-config region=' + REGION + ' -reconfigure',
        echo=True)


@task
def start_dynamo_local(c):
    "Start DynamyDB in local docker"
    c.run("docker run -d -p 8000:8000 --name dynamo amazon/dynamodb-local:latest")

