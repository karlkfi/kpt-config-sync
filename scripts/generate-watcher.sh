#!/bin/bash
#

set -euo pipefail

PROTO="third_party/googleapis/googleapis/google/watcher/v1/watch.proto"
TMP_PROTO="/tmp/watcher/watch.proto"

mkdir -p /tmp/watcher
cp $PROTO $TMP_PROTO
sed -i "s:google.golang.org/genproto/googleapis/watcher/v1;watcher:watcher/v1:g" $TMP_PROTO

protoc\
  -I /tmp/watcher \
  -I vendor/github.com/gogo/googleapis/ \
  -I vendor/ \
  -I third_party/protobuf/ \
  --gogo_out=plugins=grpc,\
Mgoogle/api/annotations.proto=github.com/gogo/googleapis/google/api,\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types,\
:clientgen \
$TMP_PROTO

mkdir -p clientgen/watcher/v1/testing

# go install github.com/golang/mock/mockgen

mockgen github.com/google/nomos/clientgen/watcher/v1 WatcherClient,Watcher_WatchClient > clientgen/watcher/v1/testing/watcher_mock.go
