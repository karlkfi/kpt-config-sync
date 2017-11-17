changecom(`<unused>')
# GEN_NOTE
kind: Config
clusters:
  - name: authorizer
    cluster:
      certificate-authority: /etc/srv/kubernetes/ca-webhook.crt
      # For now, this setup does not work.  It requires the use of an endpoint.
      # Stay tuned.
      server: https://authorizer.stolos-system.svc/authorize
users:
  - name: apiserver
    user:
      client-certificate: /etc/srv/kubernetes/pki/apiserver.crt
      client-key: /etc/srv/kubernetes/pki/apiserver.key
current-context: webhook
contexts:
  - context:
      cluster: authorizer
      user: apiserver
    name: webhook

