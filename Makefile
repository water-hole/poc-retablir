build:
	CGO_ENABLED=0 go build -ldflags="-extldflags=-static" -o crane
