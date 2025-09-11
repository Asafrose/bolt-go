build:
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/main main.go

deploy: build
	serverless deploy

remove:
	serverless remove

logs:
	serverless logs -f slack

.PHONY: build deploy remove logs
