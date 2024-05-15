#!/bin/bash
set -e

# run custom entrypoint commands
for CMD in $(compgen -v ENTRYPOINT_CMD); do
    if [ -n "${!CMD}" ] ; then
        eval "${!CMD}"
    fi
done

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

cd "${fs_var}/${cmd}" \
    && exec "${fs_app}/tcg-${cmd}"

# NOTE: do exec to replace shell process
# so application will receive OS signals
