# Build the reconciler binary
FROM golang:1.14 as builder

WORKDIR /workspace

COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o reconciler -mod=vendor cmd/reconciler/main.go

# Use distroless as minimal base image to package the reconciler binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/reconciler .
USER nonroot:nonroot

ENTRYPOINT ["/reconciler"]
