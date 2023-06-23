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

connector=$1
fs_app=/app
fs_var=/tcg

if ! [ -x "${fs_app}/${connector}/${connector}" ]; then
    echo "CONNECTOR NOT FOUND: ${fs_app}/${connector}/${connector}"
    exit 1
fi

if ! [ -f "${fs_var}/${connector}/tcg_config.yaml" ]; then
    mkdir -p "${fs_var}/${connector}"
    cp "${fs_app}/${connector}/tcg_config.yaml" "${fs_var}/${connector}/tcg_config.yaml" || exit 1
fi

cd "${fs_var}/${connector}" || exit 1

# NOTE: do exec to replace shell process
# so application will receive OS signals
exec "${fs_app}/${connector}/${connector}"
