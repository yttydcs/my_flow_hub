//go:build tools

package pb

//go:generate protoc --go_out=. --go_opt=paths=source_relative myflowhub.proto
