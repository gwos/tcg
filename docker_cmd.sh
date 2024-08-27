#!/bin/sh

# update cacerts
if ls /usr/local/groundwork/config/cacerts/*.pem > /dev/null 2>&1; then
    for CACERT in /usr/local/groundwork/config/cacerts/*.pem ; do
        CACERT_FILENAME=$(basename $CACERT)
        CACERT_NAME=${CACERT_FILENAME%.pem}
        echo "Update CA certs: adding $CACERT_FILENAME"
        openssl x509 -in $CACERT -noout -text
        cp $CACERT /usr/local/share/ca-certificates/${CACERT_NAME}.crt
    done
    update-ca-certificates
fi

# run custom entrypoint commands
for CMD_INDEX in 0 1 2 3 4 5 6 7 8 9 ; do
    CMD_VARIABLE="ENTRYPOINT_CMD_${CMD_INDEX}"
    eval CMD=\$$CMD_VARIABLE
    if [ -n "$CMD" ] ; then
        eval ${CMD}
    fi
done

# define connector app
cmd=${1//-connector/}
fs_app=/app
fs_var=/tcg

[ ! -x "${fs_app}/tcg-${cmd}" ] \
    && echo "CONNECTOR NOT FOUND: ${fs_app}/tcg-${cmd}" \
    && exit 1

[ -d "${fs_var}/${cmd}-connector" ] && [ ! -d "${fs_var}/${cmd}" ] \
    && echo "CONNECTOR COMPAT: ${cmd}-connector" \
    && ln -s "${fs_var}/${cmd}-connector" "${fs_var}/${cmd}"

[ -d "${fs_var}/kubernetes-connector" ] && [ ! -d "${fs_var}/k8s" ] \
    && echo "CONNECTOR COMPAT: k8s" \
    && ln -s "${fs_var}/kubernetes-connector" "${fs_var}/k8s"

[ ! -f "${fs_var}/${cmd}/tcg_config.yaml" ] \
    && mkdir -p "${fs_var}/${cmd}" \
    && cp "${fs_app}/${cmd}/tcg_config.yaml" "${fs_var}/${cmd}"

cd "${fs_var}/${cmd}"


# handle OS signals
#
shutdown_requested=false
handle_signal() {
    if ! "$shutdown_requested" ; then
        shutdown_requested=true
        echo "[$(date -Is)] DOCKER_CMD OS SIGNAL $1 : Exiting"
        # Terminate tcg- process and wait while it stops, then others.
        # Note, we faced different issues trying to use $pid_tcg here.
        # `tail --pid` implemented in GNU coreutils, not in busybox.
        pkill -SIGTERM -ef tcg-${cmd}
        tail --pid=$(pgrep -f tcg-${cmd}) -f /dev/null
        pkill -SIGTERM -e -s 0
    fi
}

trap "handle_signal EXIT" EXIT
trap "handle_signal HUP" HUP
trap "handle_signal INT" INT
trap "handle_signal QUIT" QUIT
trap "handle_signal KILL" KILL
trap "handle_signal TERM" TERM


# run tcg app with watchdog

TCG_RESTART_ON_CRASH=${TCG_RESTART_ON_CRASH:-true}

$TCG_RESTART_ON_CRASH &&
    echo "[$(date -Is)] DOCKER_CMD TCG_RESTART_ON_CRASH : true"
$TCG_RESTART_ON_CRASH ||
    echo "[$(date -Is)] DOCKER_CMD TCG_RESTART_ON_CRASH : false"

healthcheck() {
    TCG_CONNECTOR_CONTROLLERADDR=${TCG_CONNECTOR_CONTROLLERADDR:-127.0.0.1:8099}
    curl --fail --silent ${TCG_CONNECTOR_CONTROLLERADDR}/api/v1/identity
}

pid_tcg=-1
run_tcg() {
    ${fs_app}/tcg-${cmd} &
    pid_tcg=$!
}
run_tcg
sleep 1

while ! "$shutdown_requested" ; do
    if "$TCG_RESTART_ON_CRASH" && ! healthcheck ; then
        echo "[$(date -Is)] DOCKER_CMD TCG_RESTART_ON_CRASH : Restarting"
        kill -s SIGTERM "$pid_tcg" ; wait "$pid_tcg" ; run_tcg
        sleep 10
    fi
    sleep 10
done &


# If the wait utility is invoked with no operands, it shall wait
# until all process IDs known to the invoking shell have terminated
# and exit with a zero exit status.
#
wait
# just force 0 exit code as last command can return whatever..
exit 0
