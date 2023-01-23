set positional-arguments

my-issues: (run "-f" "./jql/my-issues.jql")

run *args:
	go run main.go "$@"

debug *args:
	dlv debug -- "$@"

build:
	go build -o jira main.go

test:
	go test ./...

lint:
	golangci-lint run
