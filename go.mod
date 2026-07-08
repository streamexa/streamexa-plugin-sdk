module github.com/streamexa/streamexa-plugin-sdk

go 1.25.0

require (
	github.com/hashicorp/go-plugin v1.8.0
	google.golang.org/grpc v1.82.0
	google.golang.org/protobuf v1.36.11
)

require google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.6.2 // indirect

require (
	github.com/fatih/color v1.13.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/oklog/run v1.1.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
)

// protoc-gen-go / protoc-gen-go-grpc are pinned here and invoked by buf.gen.yaml
// as `go tool` local plugins, so codegen versions live in go.mod.
tool (
	google.golang.org/grpc/cmd/protoc-gen-go-grpc
	google.golang.org/protobuf/cmd/protoc-gen-go
)
