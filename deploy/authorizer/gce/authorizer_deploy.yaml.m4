changecom(`<unused>')
# GEN_NOTE
apiVersion: v1
kind: Service
metadata:
        name: PACKAGE
        namespace: stolos-system
spec:
        selector:
                app: authz
        ports:
        - name: foo
          port: 443
          targetPort: https-auth-port
        clusterIP: CLUSTER_IP
---
apiVersion: v1
kind: Pod
metadata:
        name: authorizer
        namespace: stolos-system
        labels:
                app: authz
spec:
        serviceAccountName: stolos-service
        containers:
        - name: authorizer
          image: gcr.io/GCP_PROJECT/PACKAGE:IMAGE_TAG
          imagePullPolicy: Always
          ports:
                - containerPort: 8443
                  name: https-auth-port
          args: [
                  "--logtostderr",
                  "--vmodule=main=2"
                ]
        restartPolicy: Always

