#!/bin/sh
#
# To deploy importer:
#
# $ make deploy
# $ ./scripts/deploy-gcp-importer.sh

CREDS_FILE=~/watcher_client_key.json
ORG_ID=515925372711

kubectl delete deployment git-policy-importer -n nomos-system
kubectl delete configmap gcp-policy-importer -n nomos-system
kubectl delete secret gcp-creds -n nomos-system

kubectl create configmap gcp-policy-importer -n nomos-system \
    --from-literal=ORG_ID=$ORG_ID \
    --from-literal=POLICY_API_ADDRESS=autopush-kubernetespolicy.sandbox.googleapis.com:443
kubectl create secret generic gcp-creds -n nomos-system --from-file=gcp-private-key=$CREDS_FILE

make redeploy-gcp-policy-importer

