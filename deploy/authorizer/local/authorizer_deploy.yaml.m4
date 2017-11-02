apiVersion: v1
kind: Service
metadata:
        name: authorizer
        namespace: stolos-system
spec:
        selector:
                app: authz
        ports:
        - name: foo
          port: 443
          targetPort: https-auth-port
        clusterIP: 10.0.0.112
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
          image: authorizer:test
          imagePullPolicy: IfNotPresent
          ports:
                - containerPort: 8443
                  name: https-auth-port
          args: [
                  "--logtostderr",
                  "--vmodule=main=2"
                ]
        restartPolicy: Always

