# GEN_NOTE
kind: Config
clusters:
  - name: authorizer
    cluster:
      certificate-authority: WEBHOOK_CA_CERT
      server: https://authorizer.stolos-system.svc/authorize
users:
  - name: apiserver
    user:
      client-certificate: /var/run/kubernetes/server-ca.crt
      client-key: /var/run/kubernetes/server-ca.key
current-context: webhook
contexts:
  - context:
      cluster: authorizer
      user: apiserver
    name: webhook

