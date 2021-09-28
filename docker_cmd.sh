#!/bin/sh

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
