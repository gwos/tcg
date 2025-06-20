#
# NOTE:
# https://stackoverflow.com/questions/36279253/go-compiled-binary-wont-run-in-an-alpine-docker-container-on-ubuntu-host
#
FROM golang:1.24-bullseye AS build-libtransit
ARG TRAVIS_TAG=
ENV TRAVIS_TAG=${TRAVIS_TAG:-master}
WORKDIR /go/src/
COPY . .

RUN apt-get update -y \
    && echo "--" \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y \
        libjansson-dev libmcrypt-dev \
    && echo "--"

RUN go test  $(go list ./... | grep -v tcg/integration) \
    && echo "[Go TESTS DONE]"

RUN make clean && make \
    && cp libtransit/libtransit.so libtransit/libtransit.h libtransit/transit.h  build/ \
    && echo "[LIBTRANSIT BUILD DONE]"

FROM scratch AS export-libtransit
COPY --from=build-libtransit /go/src/build /

FROM golang:1.24-alpine AS build
ARG TRAVIS_TAG=
ENV TRAVIS_TAG=${TRAVIS_TAG:-master}
WORKDIR /go/src/
COPY . .

RUN apk add --no-cache \
        bash build-base git \
        libmcrypt libmcrypt-dev \
    && echo "[CHECKER NSCA DEPS DONE]"

# use bash for run to support '[[' command
SHELL ["/bin/bash", "-c"]

RUN sh -x \
    && mkdir -p /app \
    && build_time=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
    && ldflags="-X 'github.com/gwos/tcg/config.buildTime=${build_time}'" \
    && ldflags="${ldflags} -X 'github.com/gwos/tcg/config.buildTag=${TRAVIS_TAG}'" \
    && CGO_ENABLED=1 go build -v -o /app/tcg -ldflags "$ldflags" . \
    && echo "[TCG BUILD DONE]"

RUN sh -x \
    && for d in /go/src/connectors/*/ ; \
    do  [ ! -f "${d}tcg_config.yaml" ] && continue ; \
        cmd=$(basename "$d") ; dest="/app/${cmd}" ; \
        echo "__ $cmd __" ; mkdir -p "$dest" \
        && cp "${d}tcg_config.yaml" "$dest" \
        && ln -s /app/tcg "/app/tcg-${cmd}" ; \
    done \
    && echo "[CONNECTORS DONE]"

RUN sh -x \
    && [ -d /app/k8s ] && ln -s /app/k8s /app/kubernetes  \
    && ln -s /app/tcg /app/tcg-kubernetes \
    && echo "[ALIASES DONE]"

RUN cp ./docker_cmd.sh /app/

# Support custom-build-outputs for debug the build
# https://docs.docker.com/engine/reference/commandline/build/#custom-build-outputs
FROM scratch AS export
COPY --from=build /app /

FROM alpine:3.11 AS prod
# update zlib to fix CVE
RUN apk add -u --no-cache \
        bash coreutils procps \
        ca-certificates openssl \
        curl jq vim \
        libmcrypt \
        zlib \
    && update-ca-certificates
COPY --from=build /app /app

# Land docker exec into var folder
WORKDIR /tcg/
CMD ["/app/docker_cmd.sh", "apm-connector"]
