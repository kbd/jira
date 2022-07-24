package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/andygrunwald/go-jira"
	"github.com/k0kubun/pp/v3"
	"github.com/kbd/jira/util"
	"github.com/kbd/pps"
)

var CLI struct {
	Tickets bool `help:"List tickets"`
	Epics   bool `help:"list epics"`
	View    struct {
		Ids []string `arg:""`
	} `cmd:"" help:"Issue(s) to view" xor:"File,Expression,View"`
	File       string `short:"f" help:"Execute JQL expression in file" type:"path" xor:"File,Expression,View"`
	Expression string `short:"e" help:"Execute JQL expression"  xor:"File,Expression,View"`
}

type Client struct {
	Url  string
	Jira *jira.Client
}

func NewClient(url, token string) (*Client, error) {
	tp := jira.BearerAuthTransport{Token: token}
	jiraclient, err := jira.NewClient(tp.Client(), url)
	if err != nil {
		return nil, err
	}
	client := &Client{
		Url:  url,
		Jira: jiraclient,
	}
	return client, nil
}

func parseCLI() {
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
}

func main() {
	pp.SetColorScheme(pps.Scheme)
	parseCLI()

	// grab required environment variables
	jiraToken, ok := os.LookupEnv("JIRA_TOKEN")
	if !ok {
		log.Fatalf("expect JIRA_TOKEN in environment")
	}
	jiraUrl, ok := os.LookupEnv("JIRA_URL")
	if !ok {
		log.Fatal("expect JIRA_URL in environment")
	}

	// create the jira client
	client, err := NewClient(jiraUrl, jiraToken)
	if err != nil {
		log.Fatalf("couldn't create JIRA client: %s", err)
	}

	// handle command line arguments
	jql := ""
	if CLI.File != "" || CLI.Expression != "" {
		if CLI.File != "" {
			// fmt.Println("Got file: ", CLI.File)
			jqlbytes, err := os.ReadFile(CLI.File)
			if err != nil {
				log.Fatalf("couldn't read file: %s", CLI.File)
			}
			jql = string(jqlbytes)
			// fmt.Println("Got jql:", jql)
		} else if CLI.Expression != "" {
			jql = CLI.Expression
		}
		// display issues
		err = displayIssuesList(client, jql)
		if err != nil {
			log.Fatalf("error displaying issues: %s", err)
		}
	} else if len(CLI.View.Ids) > 0 {
		err := displayIssues(client, CLI.View.Ids)
		if err != nil {
			log.Fatalf(err.Error())
		}
	}
}

func displayIssues(client *Client, issues []string) error {
	for _, i := range issues {
		issue, _, err := client.Jira.Issue.Get(i, nil)
		if err != nil {
			return err
		}
		fmt.Printf("Key: %s\n", issue.Key)
		fmt.Printf("Summary: %s\n", issue.Fields.Summary)
		fmt.Printf("Desc: %s\n", issue.Fields.Description)
	}
	return nil
}

func displayIssuesList(client *Client, jql string) error {
	os.Stderr.WriteString(fmt.Sprintf("Executing jql: '%s'\n", jql))
	if jql == "" {
		return fmt.Errorf("empty JQL expression provided")
	}

	// search for issues with the provided query
	issues, _, err := client.Jira.Issue.Search(string(jql), nil)
	if err != nil {
		return fmt.Errorf("error in query: %w", err)
	}

	// show fuzzy finder for issues
	options := make([]string, len(issues))
	for i, issue := range issues {
		options[i] = fmt.Sprintf("%s ('%s'): %s", issue.Key, issue.Fields.Status.Name, issue.Fields.Summary)
	}
	result := util.Fzf(options)
	if len(result) == 0 { // nothing chosen
		os.Exit(0)
	}

	// for each result, split on the first space and take that as the value
	var keys []string
	for _, r := range result {
		keys = append(keys, strings.SplitN(r, " ", 2)[0])
	}
	issueUrl := getIssueUrlForKey(client.Url, keys[0])
	code := openBrowser(issueUrl)
	if code != 0 {
		fmt.Printf("Got return code %d from process\n", code)
	}
	return nil
}

func openBrowser(url string) (returncode int) {
	// farm out to python's cross-platform 'webbrowser' module
	fmt.Printf("Launching: %s\n", url)
	safeUrl := strings.ReplaceAll(url, "'", "\\'")
	pythonCode := fmt.Sprintf("import webbrowser as b; b.open(r'%s')", safeUrl)
	cmd := exec.Command("python3", "-c", pythonCode)
	if err := cmd.Run(); err != nil {
		return 1
	}
	return cmd.ProcessState.ExitCode()
}

func getIssueUrlForKey(jiraUrl, key string) (issueUrl string) {
	parsedUrl, err := url.Parse(jiraUrl)
	if err != nil {
		log.Fatal(err)
	}
	parsedUrl.Path = path.Join(parsedUrl.Path, "browse", key)
	return parsedUrl.String()
}
