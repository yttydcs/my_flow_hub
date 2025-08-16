module myflowhub/server

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/rs/zerolog v1.34.0
	golang.org/x/crypto v0.31.0
	gorm.io/datatypes v1.2.6
	gorm.io/gorm v1.30.1
	myflowhub/pkg/config v0.0.0
	myflowhub/pkg/database v0.0.0
	myflowhub/pkg/protocol/binproto v0.0.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	gorm.io/driver/mysql v1.5.6 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
)

replace myflowhub/pkg/config => ../pkg/config

replace myflowhub/pkg/database => ../pkg/database

replace myflowhub/pkg/protocol/binproto => ../pkg/protocol/binproto
