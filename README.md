Greeter
------

Simple grpc server / client for testing

``` shell
Usage of ./greeter:
  -address string
        GRPC endpoint address in the format host:port (client only) (default "localhost:50051")
  -client
        Run as grpc client
  -headers string
        Comma-separated list of key=value headers, e.g., 'Authorization=token,Env=prod' (client only)
  -insecure
        Use an insecure connection (client only)
```

## Deploying in k8s cluster

``` yaml
# Namespace definition
apiVersion: v1
kind: Namespace
metadata:
  name: greeter
---
# Deployment definition
apiVersion: apps/v1
kind: Deployment
metadata:
  name: greeter-deployment
  namespace: greeter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: greeter
  template:
    metadata:
      labels:
        app: greeter
    spec:
      containers:
        - name: greeter
          image: ghcr.io/michalskalski/greeter/greeter:v2
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
          ports:
            - containerPort: 50051
          readinessProbe:
            tcpSocket:
              port: 50051
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            tcpSocket:
              port: 50051
            initialDelaySeconds: 10
            periodSeconds: 20
          env:
            - name: ENV_NAME
              value: "development"
---
# Service definition
apiVersion: v1
kind: Service
metadata:
  name: greeter-service
  namespace: greeter
spec:
  selector:
    app: greeter
  ports:
    - port: 50051
      targetPort: 50051
      protocol: TCP
      name: grpc
  type: ClusterIP
---
# HTTPRoute definition
apiVersion: gateway.networking.k8s.io/v1beta1
kind: HTTPRoute
metadata:
  name: greeter-route
  namespace: greeter
spec:
  parentRefs:
    - name: gateway
      namespace: gateway
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /greeter.Greeter/
      backendRefs:
        - name: greeter-service
          port: 50051
          namespace: greeter
```
