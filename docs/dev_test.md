# Testing

## e2e

end-to-end tests deploy Nomos components on Kubernetes cluster
and verify functionality.

```shell
cd $GOROOT/src
go install github.com/google/nomos/cmd/nomos-end-to-end/
```

Run the end to end test

```shell
nomos-end-to-end -repo_dir $NOMOS
```
