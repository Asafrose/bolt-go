module aws-lambda

go 1.25.0

replace github.com/Asafrose/bolt-go => ../..

require (
	github.com/Asafrose/bolt-go v0.0.0-20250911113723-50618c94346b
	github.com/aws/aws-lambda-go v1.41.0
	github.com/slack-go/slack v0.17.3
)

require github.com/gorilla/websocket v1.5.3 // indirect
