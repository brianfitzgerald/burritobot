build:
	env GOOS=linux go build -ldflags="-s -w" -o bin/model model/model.go model/slack_key.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/event_listener event_listener/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/weekly_updater weekly_updater/main.go
