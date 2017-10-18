apiVersion: v1
kind: Service
metadata:
    name: admit-resource-quota
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
    namespace: default
    labels:
        webhook: resource-quota
spec:
    automountServiceAccountToken: true
    serviceAccountName: syncer-service
    containers:
    - name: admit-resource-quota
      image: IMAGE_NAME
      ports:
            - containerPort: 8000
      args: ["--logtostderr"]
      imagePullPolicy: Always
    restartPolicy: Always
