#!/bin/bash
#
# Creates a server certificate, signed by the certificate authority that
# Minikube trusts.  The resulting certificate can be used by any servers that
# need to be called by apiserver, for example a webhook authorizer.

# The Certificate Authority (CA) files.  By default, these are taken from a
# local minikube installation.  They will probably need to come from elsewhere
# otherwise.

## The CA private key, used for signing certificate requests.
CA_PRIVATE_KEY=${CA_PRIVATE_KEY:-"$HOME/.minikube/ca.key"}

## The CA certificate (self-signed), used for attaching additional metadata
## when signing a certificate request.
CA_CERTIFICATE=${CA_CERTIFICATE:-"$HOME/.minikube/ca.crt"}

# The server files.

## The server's public and private key pair.
SERVER_PRIVATE_KEY="server.key"
SERVER_PUBLIC_KEY="server.pem"

## The server's certificate signing request (CSR).  This CSR contains the
## metadata that are handed over to the CA for signing.  The CSR is signed by
## the server's private key.
SERVER_CERTIFICATE_SIGNING_REQUEST="server.csr"

## The server certificate is the end output of the whole process: a certificate
## file that claims the identity of the server, and carries a signature that
## says that CA has seen the certificate and testifies that it checks out.
SERVER_CERTIFICATE="server.crt"

CANONICAL_NAME=authorizer.default.svc
AUTHORIZER_CLUSTER_IP_ADDRESS=${AUTHORIZER_CLUSTER_IP_ADDRESS:-10.0.0.112}

echo "Generating server private key."
openssl genrsa -out ${SERVER_PRIVATE_KEY} 2048
echo "Generating server public key."
openssl rsa -in ${SERVER_PRIVATE_KEY} -outform PEM \
  -pubout -out ${SERVER_PUBLIC_KEY}

echo "Generating server certificate for hostname: CN=${CANONICAL_NAME}"

# First generate a certificate signing request (CSR).  It must be signed with
# the server's private key.  The CN= field must match the FQDN that will appear
# in client HTTPS requests.
cat <<EOF > csr_config.tmp
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]
countryName = Country Name (2 letter code)
countryName_default = US
stateOrProvinceName = State or Province Name (full name)
stateOrProvinceName_default = CA
localityName = Locality Name (eg, city)
localityName_default = San Francisco
organizationalUnitName = Organizational Unit Name (eg, section)
organizationalUnitName_default = Example Org Unit
commonName = Example Authorizer in a Pod.
commonName_max = 64

[ v3_req ]
# Extensions to add to a certificate request
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1=${CANONICAL_NAME}
DNS.2=authorizer.default.svc.cluster.local
IP.1=${AUTHORIZER_CLUSTER_IP_ADDRESS}

EOF
SUBJECT="/C=US/ST=California/L=San Francisco/O=ExampleDotCom/OU=ExampleOU/CN=${CANONICAL_NAME}/"
openssl req -out \
  ${SERVER_CERTIFICATE_SIGNING_REQUEST} -new \
  -config csr_config.tmp \
  -extensions v3_req \
  -key ${SERVER_PRIVATE_KEY} \
  -subj "${SUBJECT}" \

# Verify that the result makes sense.  Suppresses output since it is not
# relevant.
openssl req -verify -in ${SERVER_CERTIFICATE_SIGNING_REQUEST} -text -noout

# Sign the CSR using the CA's private key.
openssl x509 -req \
  -days 360 \
  -in ${SERVER_CERTIFICATE_SIGNING_REQUEST} \
  -CA ${CA_CERTIFICATE} \
  -CAkey ${CA_PRIVATE_KEY} \
  -CAcreateserial \
  -extensions v3_req \
  -extfile csr_config.tmp \
  -out ${SERVER_CERTIFICATE}

echo "The content of the generated certificate in file: ${SERVER_CERTIFICATE}"
openssl x509 -in ${SERVER_CERTIFICATE} -text

