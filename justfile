set positional-arguments

run *args:
	go run main.go "$@"

debug *args:
	dlv debug -- "$@"

my-issues: (run "-f" "./jql/assignee.jql")
