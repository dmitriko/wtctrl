docker build -t audiolayer -f Dockerfile.sox .
docker run -it --rm  -w /opt/ -v /tmp:/tmp audiolayer cp -r /opt /tmp/opt
mv /tmp/opt/* ../../lambda/audio/
chmod +x ../../lambda/audio/sox
rm -rf /tmp/opt
