import io
import zipfile
from pathlib import Path

from invoke import task

import boto3
from botocore.errorfactory import ClientError
import IPython


DEPLOY_BUCKET = "webtectrl-deploy"


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
def lambda_deploy(c, path, bucket=DEPLOY_BUCKET):
    "Build golang lambda, zip it and upload to deploy bucket"
    s3 = boto3.client('s3')
    bin_name = "main"  # name of binary file
    path = Path(path)
    s3key = "{}/{}.zip".format(path.parts[-2], path.parts[-1])
    s3key_prev = "{}/{}.prev.zip".format(path.parts[-2], path.parts[-1])
    s3_backup(bucket, s3key, s3key_prev)
    with c.cd(str(path)):
        c.run("go get .")
        c.run("GOOS=linux GOARCH=amd64 go build -o {}".format(bin_name))
        bin_path = path / bin_name
        zip_buf = io.BytesIO()
        with zipfile.ZipFile(zip_buf, 'w', zipfile.ZIP_DEFLATED, False) as zf:
            zf.writestr(bin_name, bin_path.read_bytes())
        zip_buf.seek(0)
        s3.upload_fileobj(zip_buf, bucket, s3key)
        c.run('unlink {}'.format(bin_name))

