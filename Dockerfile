FROM ubuntu:bionic

# Choose apt ubuntu mirror
RUN sed -i -e 's@http://archive.ubuntu.com/ubuntu/@mirror://mirrors.ubuntu.com/mirrors.txt@' /etc/apt/sources.list

RUN apt-get update -qq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -qqy \
        software-properties-common \
        build-essential \
        libjansson-dev \
    && add-apt-repository ppa:longsleep/golang-backports \
    && apt-get install -qqy \
        golang-go

WORKDIR /src/

COPY . /src/

# CMD ./docker_cmd.sh
