package main

import (
	"fmt"
	"log"
	"os"

	jira "github.com/andygrunwald/go-jira"
	"github.com/k0kubun/pp/v3"
)

func main() {
	// grab required environment variables
	token, ok := os.LookupEnv("JIRA_TOKEN")
	if !ok {
		fmt.Println("expect JIRA_TOKEN in environment")
		os.Exit(1)
	}

	url, ok := os.LookupEnv("JIRA_URL")
	if !ok {
		log.Fatal("expect JIRA_URL in environment", 1)
	}

	fmt.Printf("token: %s, url: %s\n", token, url)

	// create the jira client
	tp := jira.BearerAuthTransport{
		Token: token,
	}
	client, err := jira.NewClient(tp.Client(), url)
	if err != nil {
		log.Fatalf("couldn't create JIRA client: %s", err)
	}

	// run a method
	// parse command line arguments and call whatever method through reflection
	jql := "assignee=kdevens"
	issues, _, err := client.Issue.Search(jql, nil)
	if err != nil {
		log.Fatalf("error in query: %s", err)
	}
	// fmt.Printf("%#v", issues)
	pp.Print(issues[0].Key, issues[0].Fields.Summary)
}
