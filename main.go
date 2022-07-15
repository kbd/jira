package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/alecthomas/kong"
	"github.com/andygrunwald/go-jira"
	"github.com/k0kubun/pp/v3"
	"github.com/kbd/pps"
)

func main() {
	pp.SetColorScheme(pps.Scheme)

	var CLI struct {
		Tickets    bool   `help:"List tickets"`
		Epics      bool   `help:"list epics"`
		File       string `short:"f" help:"Execute JQL expression in file" type:"path"`
		Expression string `short:"e" help:"Execute JQL expression"`
	}

	ctx := kong.Parse(&CLI)
	if ctx.Empty() {
		if err := ctx.PrintUsage(true); err != nil {
			log.Fatalf(err.Error())
		}
		os.Exit(0)
	} else {
		if err := ctx.Validate(); err != nil {
			if err := ctx.PrintUsage(true); err != nil {
				log.Fatalf(err.Error())
			}
			log.Fatalf(err.Error())
		}
	}

	// grab required environment variables
	token, ok := os.LookupEnv("JIRA_TOKEN")
	if !ok {
		log.Fatalf("expect JIRA_TOKEN in environment")
	}
	url, ok := os.LookupEnv("JIRA_URL")
	if !ok {
		log.Fatal("expect JIRA_URL in environment")
	}

	fmt.Printf("token: %s..., url: %s\n\n", token[0:5], url)
	pp.Println(CLI)

	// run a method
	// parse command line arguments and call whatever method through reflection
	jql := ""
	if CLI.File != "" {
		fmt.Println("Got file: ", CLI.File)
		jqlbytes, err := os.ReadFile(CLI.File)
		if err != nil {
			log.Fatalf("couldn't read file: %s", CLI.File)
		}
		jql = string(jqlbytes)
		fmt.Println("Got jql:", jql)
	}

	// create the jira client
	tp := jira.BearerAuthTransport{Token: token}
	client, err := jira.NewClient(tp.Client(), url)
	if err != nil {
		log.Fatalf("couldn't create JIRA client: %s", err)
	}

	issues, _, err := client.Issue.Search(string(jql), nil)
	if err != nil {
		log.Fatalf("error in query: %s", err)
	}

	options := make([]string, len(issues))
	for i, issue := range issues {
		options[i] = fmt.Sprintf("%s ('%s'): %s", issue.Key, issue.Fields.Status.Name, issue.Fields.Summary)
	}

	fzf(options)
}

func fzf(options []string) (result []string) {
	nullbyte := []byte{0}

	// execute fzf, pass it options, return the selected options
	cmd := exec.Command("fzf", "--read0", "--print0")
	cmd.Stderr = os.Stderr // fzf displays its interface over stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	// asynchronously write stdin options to fzf
	go func() {
		for _, o := range options {
			// todo: ignore io errors for now, crashing is fine
			stdin.Write([]byte(o))
			stdin.Write(nullbyte)
		}
		stdin.Close()
	}()

	// get fzf result
	out, err := cmd.Output()
	if err != nil {
		if err.(*exec.ExitError).ExitCode() == 130 {
			return result // 130 is intentional exit from fzf, return nothing
		}
		log.Fatal(err)
	}

	// trim off the trailing null from fzf before splitting
	out = bytes.TrimRight(out, "\x00")
	for _, line := range bytes.Split(out, nullbyte) {
		result = append(result, string(line))
	}
	pp.Print(result)
	return result
}
