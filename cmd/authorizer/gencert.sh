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

CANONICAL_NAME=localhost

echo "Generating server private key."
openssl genrsa -out ${SERVER_PRIVATE_KEY} 2048
echo "Generating server public key."
openssl rsa -in ${SERVER_PRIVATE_KEY} -outform PEM \
  -pubout -out ${SERVER_PUBLIC_KEY}

echo "Generating server certificate for hostname: CN=${CANONICAL_NAME}"

# First generate a certificate signing request (CSR).  It must be signed with
# the server's private key.  The CN= field must match the FQDN that will appear
# in client HTTPS requests.
SUBJECT="/C=US/ST=California/L=San Francisco/O=ExampleDotCom/OU=ExampleOU/CN=${CANONICAL_NAME}/"
openssl req -out \
  ${SERVER_CERTIFICATE_SIGNING_REQUEST} -new \
  -key ${SERVER_PRIVATE_KEY} \
  -subj "${SUBJECT}"

# Verify that the result makes sense.  Suppresses output since it is not
# relevant.
openssl req -verify -in ${SERVER_CERTIFICATE_SIGNING_REQUEST} -text -noout

# Sign the CSR using the CA's private key.
openssl x509 -req -days 360 -in ${SERVER_CERTIFICATE_SIGNING_REQUEST} \
  -CA ${CA_CERTIFICATE} -CAkey ${CA_PRIVATE_KEY} \
  -CAcreateserial -out ${SERVER_CERTIFICATE}

echo "The content of the generated certificate:"
openssl x509 -in ${SERVER_CERTIFICATE} -text

