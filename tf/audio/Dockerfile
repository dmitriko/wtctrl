FROM amazonlinux:2
RUN amazon-linux-extras install epel -y && \
  yum install -y  \
    wget libogg-devel opus-devel opusfile-devel bzip2 libgomp tar \
    gcc-c++ make lame-devel zip

RUN wget -O /tmp/sox-14.4.2.tar.bz2 "http://downloads.sourceforge.net/project/sox/sox/14.4.2/sox-14.4.2.tar.bz2?r=http%3A%2F%2Fsourceforge.net%2Fprojects%2Fsox%2Ffiles%2Fsox%2F14.4.2%2F&ts=1416316415&use_mirror=heanet"
RUN cd /tmp && tar xjvf sox-14.4.2.tar.bz2 && cd /tmp/sox-14.4.2 && \
    ./configure --with-lame --with-opus --disable-shared --enable-static && make -s && make install
RUN mkdir -p /opt/lib && cp /usr/local/bin/sox  /opt && \
    cp /usr/lib64/libopusfile.so.0 /opt/lib && \
    cp /usr/lib64/libogg.so.0 /opt/lib && \
    cp /usr/lib64/libgomp.so.1 /opt/lib && \
    cp /usr/lib64/libopus.so.0 /opt/lib && \
    cp /usr/lib64/libmp3lame.so.0 /opt/lib 
