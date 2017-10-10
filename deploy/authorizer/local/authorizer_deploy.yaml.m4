apiVersion: v1
kind: Service
metadata:
        name: authorizer
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
        labels:
                app: authz
spec:
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

