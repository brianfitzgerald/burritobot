build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/model model/model.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/event_listener event_listener/main.go
