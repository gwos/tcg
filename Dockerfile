FROM ubuntu:bionic

ARG GITHUB_TOKEN

# Choose apt ubuntu mirror
RUN sed -i -e 's@http://archive.ubuntu.com/ubuntu/@mirror://mirrors.ubuntu.com/mirrors.txt@' /etc/apt/sources.list

RUN apt-get update -qq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -qqy \
        software-properties-common \
        build-essential \
        wget \
        unzip \
        maven \
        openjdk-8-jdk

RUN add-apt-repository ppa:longsleep/golang-backports \
    && apt-get update -qq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -qqy \
        golang-go

WORKDIR /tmp
RUN wget --progress=bar:force --header="Authorization: token ${GITHUB_TOKEN}" -O master.zip https://api.github.com/repos/gwos/debian-c-packages/zipball \
    && unzip -qq master.zip \
    && rm -rf master.zip
RUN apt-get install -qqy \
        ./gwos-debian-c-packages-*/jansson/libjansson4_2.12-1_amd64.deb \
        ./gwos-debian-c-packages-*/jansson/libjansson-dev_2.12-1_amd64.deb \
        ./gwos-debian-c-packages-*/jansson/libjansson-doc_2.12-1_all.deb \
    && rm -rf ./gwos-debian-c-packages-*

# https://github.com/carlossg/docker-maven/tree/master/jdk-8

WORKDIR /src/

COPY . /src/

RUN go get -t ./...
RUN go build -buildmode=c-shared -o libtransit/libtransit.so libtransit/libtransit.go

CMD ./docker_cmd.sh
