module myflowhub/client

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
)

require myflowhub/poc/protocol v0.0.0-00010101000000-000000000000 // indirect

replace myflowhub/poc/protocol => ../protocol
