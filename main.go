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
	Op      struct {
		Issue      []string `arg:"" optional:"" help:"Issue(s) to view"`
		File       string   `short:"f" help:"Execute JQL expression in file" type:"path"`
		Expression string   `short:"e" help:"Execute JQL expression"`
	} `embed:"" default:"" cmd:"" required:"" xor:"File,Expression,Issue"`
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
	if CLI.Op.File != "" || CLI.Op.Expression != "" {
		if CLI.Op.File != "" {
			jqlbytes, err := os.ReadFile(CLI.Op.File)
			if err != nil {
				log.Fatalf("couldn't read file: %s", CLI.Op.File)
			}
			jql = string(jqlbytes)
		} else if CLI.Op.Expression != "" {
			jql = CLI.Op.Expression
		}
		// display issues
		err = displayIssuesList(client, jql)
		if err != nil {
			log.Fatalf("error displaying issues: %s", err)
		}
	} else if len(CLI.Op.Issue) > 0 {
		err := displayIssues(client, CLI.Op.Issue)
		if err != nil {
			log.Fatalf(err.Error())
		}
	} else {
		os.Stderr.WriteString("No action")
		os.Exit(1)
	}
}

func displayIssues(client *Client, issues []string) error {
	// fmt.Printf("Searching for issues: %#v\n", issues)
	for _, i := range issues {
		issue, _, err := client.Jira.Issue.Get(i, nil)
		if err != nil {
			return err
		}
		fmt.Printf("Key: %s\n", issue.Key)
		fmt.Printf("\nSummary: %s\n", issue.Fields.Summary)
		fmt.Printf("\nDesc:\n%s\n", issue.Fields.Description)

		subtasks := issue.Fields.Subtasks
		if len(subtasks) > 0 {
			fmt.Printf("\nSubtasks:\n")
			for _, s := range subtasks {
				fmt.Printf("%s: %s\n", s.Fields.Summary, s.Fields.Description)
			}
		}

		comments := issue.Fields.Comments.Comments
		if len(comments) > 0 {
			fmt.Printf("\nComments:\n")
			for _, c := range comments {
				fmt.Printf("%s: %s\n", c.Author.Name, c.Body)
			}
		}
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
