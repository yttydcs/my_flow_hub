module myflowhub/pkg/protocol/binproto

go 1.23

toolchain go1.24.5

require (
	google.golang.org/protobuf v1.36.9
	myflowhub/pkg/protocol v0.0.0
)

replace myflowhub/pkg/protocol => ../
