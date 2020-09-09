#!/usr/bin/env bash

set -euo pipefail

trap 'kill "$(pidof mosquitto)"; rm -rf mosquitto.conf test-certs/ 2> /dev/null' EXIT

mkdir -p test-certs

# create a test Certification Authority
openssl req \
    -new \
    -newkey ec:<(openssl ecparam -name prime256v1) \
    -days 7 \
    -nodes \
    -x509 \
    -subj "/C=IT/ST=Piemonte/L=Torino/O=TestCA/CN=CA" \
    -keyout test-certs/test-ca.key \
    -out test-certs/test-ca.crt

# install the CA
chmod 0644 test-certs/test-ca.crt
cp test-certs/test-ca.crt /usr/local/share/ca-certificates/
update-ca-certificates

# create server certificate
openssl req \
    -new \
    -newkey ec:<(openssl ecparam -name prime256v1) \
    -nodes \
    -keyout test-certs/test-server.key \
    -out test-certs/test-server.csr \
    -subj "/C=IT/ST=Piemonte/L=Torino/O=TestServer/CN=localhost"

openssl x509 \
    -req \
    -in test-certs/test-server.csr \
    -CA test-certs/test-ca.crt \
    -CAkey test-certs/test-ca.key \
    -CAcreateserial \
    -out test-certs/test-server.crt \
    -days 7 \
    -sha256

# create client certificate
openssl req \
    -new \
    -newkey ec:<(openssl ecparam -name prime256v1) \
    -nodes \
    -keyout test-certs/test-client.key \
    -out test-certs/test-client.csr \
    -subj "/C=IT/ST=Piemonte/L=Torino/O=TestServer/CN=localhost"

openssl x509 \
    -req \
    -in test-certs/test-client.csr \
    -CA test-certs/test-ca.crt \
    -CAkey test-certs/test-ca.key \
    -CAcreateserial \
    -out test-certs/test-client.crt \
    -days 7 \
    -sha256

# generate mosquitto config to use client certificates
echo \
"port 8883
require_certificate true
cafile ./test-certs/test-ca.crt
keyfile ./test-certs/test-server.key
certfile ./test-certs/test-server.crt
log_dest none" \
> mosquitto.conf

mosquitto -c mosquitto.conf > /dev/null &

# see https://github.com/golang/go/issues/39568 about why we need to set that GODEBUG value
GODEBUG=x509ignoreCN=0 go test -v --tags=register --run="TestRegister" --timeout=15s