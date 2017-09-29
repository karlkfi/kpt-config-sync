#!/bin/bash
#
# Sets up the registration yaml for the webhook admission controller and registers it with kubectl
# Assumes gencert.sh has already been run to generate a CA certificate.

CA_BUNDLE=$(base64 --wrap=0 ca.crt)
cat > registration.yaml << EOF
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: ExternalAdmissionHookConfiguration
metadata:
  name: admit-resource-quota-reg
externalAdmissionHooks:
  - name: resourcequota.external.io
    rules:
      - operations:
          - CREATE
          - DELETE
        apiGroups:
          - "*"
        apiVersions:
          - "*"
        resources:
          - "*"
    failurePolicy: Ignore
    clientConfig:
      service:
        namespace: default
        name: admit-resource-quota
      caBundle: ${CA_BUNDLE}
EOF

kubectl delete externaladmissionhookconfiguration admit-resource-quota-reg
sleep 1
kubectl create -f registration.yaml
rm registration.yaml
