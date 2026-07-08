# Pass --build-arg SUPPRESS_TEST=1 to skip unit tests (tests run by default)
ARG SUPPRESS_TEST=
ARG GOLANG_ALPINE=golang:1-alpine3.23
ARG GOLANG_DEBIAN=golang:1-bookworm

# branches may be updated on release
ARG GW8BASE_BRANCH=5.5.5

ARG BASE_IMG=groundworkdevelopment/gw8base-alpine:${GW8BASE_BRANCH}
# NOTE:
# https://stackoverflow.com/questions/36279253/go-compiled-binary-wont-run-in-an-alpine-docker-container-on-ubuntu-host

###############################################################################
# download and cache once depend on deps only
FROM ${GOLANG_ALPINE} AS deps
WORKDIR /go/src/
COPY go.mod go.sum ./
COPY sdk/go.mod ./sdk/
RUN go mod download

###############################################################################
# run unit tests (pass --build-arg SUPPRESS_TEST=1 to skip)
# build libtransit - 1st check for Datageyser compatibility
FROM ${GOLANG_DEBIAN} AS build-libtransit-tests

RUN set -eux ;\
    apt-get update -y ;\
    DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends -y \
        libjansson-dev libmcrypt-dev ;\
    rm -rf /var/lib/apt/lists/*

WORKDIR /go/src/
COPY . .
COPY --from=deps /go/pkg/mod /go/pkg/mod

ARG SUPPRESS_TEST
RUN [ -n "$SUPPRESS_TEST" ] || go test  $(go list ./... | grep -v tcg/integration) \
    && echo "[Go TESTS DONE]"

ARG BUILD_TAG
ARG BUILD_TIME
ARG COMMIT_HASH
ARG TRAVIS_TAG
RUN set -eux ;\
    mkdir -p dist ;\
    make clean && make ;\
    cp libtransit/libtransit.so libtransit/libtransit_compat.h libtransit/libtransit.h libtransit/sdktransit.h  dist/ ;\
    echo "[LIBTRANSIT BUILD DONE]"

###############################################################################
# build connectors
FROM ${GOLANG_ALPINE} AS build

WORKDIR /go/src/
COPY . .
COPY --from=deps /go/pkg/mod /go/pkg/mod

RUN apk add --no-cache \
        bash build-base git \
        libmcrypt libmcrypt-dev \
    && echo "[CHECKER NSCA DEPS DONE]"

SHELL ["/bin/bash", "-c"]

ARG BUILD_TAG
ARG BUILD_TIME
ARG COMMIT_HASH
ARG TRAVIS_TAG
RUN set -eux ;\
    mkdir -p /app ;\
    BUILD_TAG="${BUILD_TAG:-${TRAVIS_TAG:-${COMMIT_HASH:-8.x}}}" ;\
    BUILD_TIME="${BUILD_TIME:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}" ;\
    ldflags="-X 'github.com/gwos/tcg/config.buildTime=${BUILD_TIME}'" ;\
    ldflags="${ldflags} -X 'github.com/gwos/tcg/config.buildTag=${BUILD_TAG}'" ;\
    CGO_ENABLED=1 go build -v -o /app/tcg -ldflags "$ldflags" . ;\
    echo "[TCG BUILD DONE]"

RUN set -eux ;\
    for d in /go/src/connectors/*/ ;\
    do  [ ! -f "${d}tcg_config.yaml" ] && continue ;\
        cmd=$(basename "$d") ; dest="/app/${cmd}" ;\
        echo "__ $cmd __" ; mkdir -p "$dest" ;\
        cp "${d}tcg_config.yaml" "$dest" ;\
        ln -s /app/tcg "/app/tcg-${cmd}" ;\
    done ;\
    echo "[CONNECTORS DONE]"

RUN set -eux ;\
    [ -d /app/k8s ] && ln -s /app/k8s /app/kubernetes ;\
    ln -s /app/tcg /app/tcg-kubernetes ;\
    echo "[ALIASES DONE]"

###############################################################################
# trigger all builds while required by prod
# target for Nagios build
FROM scratch AS dist
COPY --from=build-libtransit-tests /go/src/dist /dist
COPY --from=build-libtransit-tests /go/src/build/libtransitjson.so /dist/
COPY --from=build-libtransit-tests /go/src/gotocjson/_c_code/convert_go_to_c.h /dist/
COPY --from=build-libtransit-tests /go/src/build/generic_datatypes.h /dist/
COPY --from=build-libtransit-tests /go/src/build/time.h /dist/
COPY --from=build-libtransit-tests /go/src/build/milliseconds.h /dist/
# Ship the gotocjson-generated build/transit.h as dist/transit.h: it defines
# the transit_* types and make_empty_*/free_* helpers the DataGeyser C code
# consumes. The libtransit constants header is shipped separately as
# sdktransit.h (see the dist/ copy above), so there is no name collision.
COPY --from=build-libtransit-tests /go/src/build/transit.h /dist/
COPY --from=build /app /app

###############################################################################
FROM ${BASE_IMG} AS prod
ARG BASE_IMG
COPY --from=dist /app /app
COPY docker_cmd.d/. /docker_cmd.d/
COPY docker_cmd.sh /docker_cmd.sh

RUN set -eux ;\
    # gw8base-alpine already ships the /docker_cmd.* entrypoint deps
    # (bash, ca-certificates, curl, procps). When building on any OTHER base —
    # e.g. a public alpine for 3rd-party builds that have no access to the
    # private gw8base image — install them so docker_cmd.sh (bash), the signal
    # handler (procps pkill) and the wrapper healthcheck (curl) work.
    case "$BASE_IMG" in \
        groundworkdevelopment/gw8base-alpine*) ;; \
        *) apk add --no-cache bash ca-certificates curl procps ;; \
    esac ;\
    apk add --no-cache libmcrypt ;\
    # Back-compat alias for docker-compose files in gwos/gw8 that pass
    # /app/docker_cmd.sh as entrypoint.
    ln -s /docker_cmd.sh /app/docker_cmd.sh

ARG BRANCH
ARG COMMIT_HASH
ARG TRAVIS_BUILD_ID
ARG TRAVIS_COMMIT
ARG TRAVIS_COMMIT_MESSAGE
ARG TRAVIS_JOB_ID
ARG TRAVIS_JOB_WEB_URL
ARG TRAVIS_TAG

ENV BRANCH="${BRANCH}"
ENV COMMIT_HASH="${COMMIT_HASH}"
ENV TRAVIS_BUILD_ID="${TRAVIS_BUILD_ID}"
ENV TRAVIS_COMMIT="${TRAVIS_COMMIT}"
ENV TRAVIS_COMMIT_MESSAGE="${TRAVIS_COMMIT_MESSAGE}"
ENV TRAVIS_JOB_ID="${TRAVIS_JOB_ID}"
ENV TRAVIS_JOB_WEB_URL="${TRAVIS_JOB_WEB_URL}"
ENV TRAVIS_TAG="${TRAVIS_TAG}"

# Land docker exec into var folder
WORKDIR /tcg/
CMD ["/docker_cmd.sh", "apm-connector"]
