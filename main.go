package main

import (
	"fmt"
	"log"
	"os"

	jira "github.com/andygrunwald/go-jira"
)

func main() {
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

	tp := jira.BearerAuthTransport{
		Token: token,
	}
	client, err := jira.NewClient(tp.Client(), url)
	if err != nil {
		log.Fatalf("couldn't create JIRA client: %s", err)
	}

	u, _, err := client.User.GetSelf()
	if err != nil {
		log.Fatalf("couldn't get user: %s", err)
	}
	fmt.Printf("\nEmail: %v\nSuccess!\n", u.EmailAddress)
}
