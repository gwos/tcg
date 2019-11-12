FROM ubuntu:bionic

# Choose apt ubuntu mirror
RUN sed -i -e 's@http://archive.ubuntu.com/ubuntu/@mirror://mirrors.ubuntu.com/mirrors.txt@' /etc/apt/sources.list

RUN apt-get update -qq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -qqy \
        software-properties-common \
        build-essential \
        libjansson-dev \
        maven \
        openjdk-8-jdk

RUN add-apt-repository ppa:longsleep/golang-backports \
    && apt-get update -qq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -qqy \
        golang-go

# https://github.com/carlossg/docker-maven/tree/master/jdk-8

WORKDIR /src/

COPY . /src/

# CMD ./docker_cmd.sh
