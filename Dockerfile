#
# NOTE:
# https://stackoverflow.com/questions/36279253/go-compiled-binary-wont-run-in-an-alpine-docker-container-on-ubuntu-host
#
FROM golang:latest as test-deb

WORKDIR /go/src/
COPY . .

RUN apt-get update -qq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -qqy \
        libjansson-dev \
    && make clean && make \
    && echo "[GOTOCJSON TEST DONE]"

FROM golang:alpine as build
ARG TRAVIS_TAG=
ENV TRAVIS_TAG=${TRAVIS_TAG:-master}
WORKDIR /go/src/
COPY . .

RUN apk add --no-cache \
        bash build-base \
        libmcrypt libmcrypt-dev \
    && echo "[CHECKER NSCA DEPS DONE]"

RUN go test -v $(go list ./... | grep -v tcg/integration) \
    && echo "[Go TESTS DONE]"

# use bash for run to support '[[' command
SHELL ["/bin/bash", "-c"]
RUN sh -x \
    && build_time=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
    && ldflags="-X 'github.com/gwos/tcg/config.buildTime=${build_time}'" \
    && ldflags="${ldflags} -X 'github.com/gwos/tcg/config.buildTag=${TRAVIS_TAG}'" \
    && for d in ./connectors/*connector/; \
    do  cd "$d"; pwd; \
        CGO_ENABLED=0; \
        if [[ "$d" == *nsca* ]] ; then CGO_ENABLED=1; fi; \
        echo "CGO_ENABLED:$CGO_ENABLED"; \
        CGO_ENABLED=$CGO_ENABLED go build -ldflags "$ldflags" . \
        && name=$(ls *connector) \
        && dest="/app/${name}" \
        && mkdir -p "$dest" \
        && cp *connector *config.yaml "$dest" \
        && cd -; \
    done \
    && echo "[CONNECTORS DONE]"
RUN cp ./docker_cmd.sh /app/

# Support custom-build-outputs for debug the build
# https://docs.docker.com/engine/reference/commandline/build/#custom-build-outputs
FROM scratch as export
COPY --from=build /app .

FROM alpine:3.11 as prod
RUN apk add --no-cache libmcrypt
COPY --from=build /app /app

# Land docker exec into var folder
WORKDIR /tcg/
CMD ["/app/docker_cmd.sh", "apm-connector"]
