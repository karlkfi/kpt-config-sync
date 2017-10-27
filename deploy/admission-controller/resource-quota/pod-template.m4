apiVersion: v1
kind: Service
metadata:
    name: admit-resource-quota
    namespace: stolos-system
spec:
    selector:
        webhook: resource-quota
    ports:
    - port: 443
      targetPort: 8000
---
apiVersion: v1
kind: Pod
metadata:
    name: admit-resource-quota
    namespace: stolos-system
    labels:
        webhook: resource-quota
spec:
    automountServiceAccountToken: true
    serviceAccountName: stolos-service
    containers:
    - name: admit-resource-quota
      image: IMAGE_NAME
      ports:
            - containerPort: 8000
      args: ["--logtostderr"]
      imagePullPolicy: Always
    restartPolicy: Always
