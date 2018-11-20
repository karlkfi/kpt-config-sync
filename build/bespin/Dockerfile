# Build the terraform google provider
FROM k8s.gcr.io/debian-base:0.4.0 AS terraform-bundler-build

RUN apt-get update && apt-get install -y ca-certificates

WORKDIR /bespin/

COPY .output/bespin/bin/bespin-controllers-manager .
COPY .output/bespin/terraform-bundle/ terraform-bundle/

# License file required for on-prem release.
COPY LICENSE LICENSE

ENTRYPOINT ["./bespin-controllers-manager"]
