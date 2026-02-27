#! /usr/bin/env bash
set -e

trap 'catch $?' EXIT

catch () {
    if [[ "$1" != "0" ]]; then
        echo "Failed"
        exit $1
    fi
}

project_root=$(cd $(dirname $0)/.. && pwd)

mockery

golangci-lint run
echo "Generated mocks and ran linters"

protoc -I ${project_root}/api  ${project_root}/api/*.proto --go_out=${project_root} --go-grpc_out=${project_root}
echo "Generated gRPC code"

tls_dir="${project_root}/build/example_configs/testdata"
# 1. Create CA Key and Certificate
openssl req -x509 -newkey rsa:4096 -keyout ${tls_dir}/ca.key -out ${tls_dir}/ca.crt -sha256 -days 3650 -nodes -subj "/C=US/ST=Texas/L=Houston/O=RadAF/CN=NojYerac CA"
# 2. Generate Server Key
openssl genrsa -out ${tls_dir}/priv.key 2048
# 3. Create Certificate Signing Request (CSR)
openssl req -new -key ${tls_dir}/priv.key -out ${tls_dir}/pub.csr -subj "/CN=localhost" -addext "subjectAltName = DNS:localhost"
# 4. Sign with CA
openssl x509 -req -in ${tls_dir}/pub.csr -CA ${tls_dir}/ca.crt -CAkey ${tls_dir}/ca.key -CAcreateserial -out ${tls_dir}/pub.crt -days 3650
echo "Generated TLS certificates"
