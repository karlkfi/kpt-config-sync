#!/bin/bash
#
# Creates a new certificate authority, and a server certificate, signed by that authority.
# The server certificate will include the subject CN=<name of service> to allow the admission controller
# to be called by the api server.

# The name of the service (and the namespace it runs in).
CN_BASE="admit-resource-quota.stolos-system.svc"

# Create a certificate authority
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -days 100000 -out ca.crt -subj "/CN=${CN_BASE}_ca"

# Create a server certiticate
openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -subj "/CN=${CN_BASE}" -config server-cert.conf
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 100000 -extensions v3_req -extfile server-cert.conf

rm *.csr
rm *.srl
