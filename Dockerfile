#
# NOTE:
# https://stackoverflow.com/questions/36279253/go-compiled-binary-wont-run-in-an-alpine-docker-container-on-ubuntu-host
#
FROM golang:latest as build

ARG TRAVIS_TAG=
ENV TRAVIS_TAG=${TRAVIS_TAG:-master}

WORKDIR /go/src/
COPY . .
RUN go test -v $(go list ./... | grep -v tcg/integration)
RUN sh -x \
    && build_time=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
    && ldflags="-X 'github.com/gwos/tcg/config.buildTime=${build_time}'" \
    && ldflags="${ldflags} -X 'github.com/gwos/tcg/config.buildTag=${TRAVIS_TAG}'" \
    && for d in ./connectors/*connector/; \
    do \
        cd "$d"; pwd; \
        CGO_ENABLED=0 go build -ldflags "$ldflags" . \
        && name=$(ls *connector) \
        && dest="/app/${name}" \
        && mkdir -p "$dest" \
        && cp *connector *config.yaml "$dest" \
        && cd -; \
    done \
    && mkdir -p /tcg/snmp-connector/utils \
    && cp connectors/snmp-connector/utils/xorp.pl /tcg/snmp-connector/utils \
    && apt-get update && apt-get install perl \
    && echo "[CONNECTORS DONE]"
RUN cp ./docker_cmd.sh /app/

# Support custom-build-outputs for debug the build
# https://docs.docker.com/engine/reference/commandline/build/#custom-build-outputs
FROM scratch as export
COPY --from=build /app .

FROM alpine:3.11 as prod
COPY --from=build /app /app

# Land docker exec into var folder
WORKDIR /tcg/
CMD ["/app/docker_cmd.sh", "apm-connector"]
