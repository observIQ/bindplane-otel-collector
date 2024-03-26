## How to Generate Protos

download and install protoc:
https://google.github.io/proto-lens/installing-protoc.html

Use this command where protobuf `../googleapis` is a path to [this repo]("https://github.com/googleapis/googleapis") on your local system:
```
protoc --proto_path=./exporter/chronicleexporter/protos \
       --go-grpc_opt=paths=source_relative \
       --go-grpc_out=./exporter/chronicleexporter/protos/api \
       --go_out=./exporter/chronicleexporter/protos/api \
       --go_opt=paths=source_relative \
       --proto_path=../googleapis \
       ./exporter/chronicleexporter/protos/*.proto
```