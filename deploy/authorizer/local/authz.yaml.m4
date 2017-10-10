# GEN_NOTE
kind: Config
clusters:
  - name: authorizer
    cluster:
      certificate-authority: /usr/local/google/home/fmil/.minikube/ca.crt
      server: https://authorizer.default.svc/authorize
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

