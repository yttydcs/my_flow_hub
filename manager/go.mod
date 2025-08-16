module myflowhub/manager

go 1.21

require (
	github.com/gorilla/websocket v1.5.3
	github.com/rs/zerolog v1.34.0
	myflowhub/pkg/config v0.0.0-00010101000000-000000000000
	myflowhub/pkg/protocol/binproto v0.0.0-00010101000000-000000000000
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/sys v0.28.0 // indirect
)

replace myflowhub/pkg/config => ../pkg/config

replace myflowhub/pkg/database => ../pkg/database

replace myflowhub/pkg/protocol/binproto => ../pkg/protocol/binproto
