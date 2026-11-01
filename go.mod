module github.com/bitoolkit/biternet-go

go 1.27.3

tool github.com/egoist/mygo/cmd/mygo

require (
	github.com/bitoolkit/biternet-go/client v0.0.0
	github.com/egoist/mygo v0.7.4
	golang.org/x/mod v0.41.0
)

require (
	github.com/ebitengine/purego v0.11.1 // indirect
	github.com/go-text/typesetting v0.3.5 // indirect
	golang.org/x/image v0.46.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)

replace github.com/bitoolkit/biternet-go/client => ./client
