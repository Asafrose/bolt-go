module socket-mode

go 1.25.0

replace github.com/Asafrose/bolt-go => ../..

require github.com/Asafrose/bolt-go v0.0.0-20250911113723-50618c94346b

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/samber/lo v1.51.0
	github.com/slack-go/slack v0.17.3
	golang.org/x/text v0.22.0 // indirect
)
