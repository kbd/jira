package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/alecthomas/kong"
	"github.com/andygrunwald/go-jira"
	"github.com/charmbracelet/glamour"
	"github.com/k0kubun/pp/v3"
	"github.com/kbd/jira/pkg/util"
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

func NewClient(url, user, token string) (*Client, error) {
	tp := jira.BasicAuthTransport{
		Username: user,
		Password: token,
	}
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
	jiraUser, ok := os.LookupEnv("JIRA_USER")
	if !ok {
		log.Fatal("expect JIRA_USER in environment")
	}

	// create the jira client
	client, err := NewClient(jiraUrl, jiraUser, jiraToken)
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
			jql = strings.TrimSpace(string(jqlbytes))
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
		os.Stderr.WriteString("No action\n")
		os.Exit(1)
	}
}

func displayIssue(client *Client, issueKey string) error {
	converter := md.NewConverter("", true, nil)
	buffer := strings.Builder{}

	opts := jira.GetQueryOptions{
		Expand: "renderedFields",
	}
	issue, _, err := client.Jira.Issue.Get(issueKey, &opts)
	if err != nil {
		return err
	}
	buffer.WriteString(fmt.Sprintf("# %s: %s\n", issue.Key, issue.Fields.Summary))

	// convert Jira's HTML rendered from ADF to markdown
	desc := issue.RenderedFields.Description
	desc, err = converter.ConvertString(desc)
	if err != nil {
		return fmt.Errorf("couldn't convert HTML to markdown: %w", err)
	}
	buffer.WriteString(desc)
	buffer.WriteString("\n")

	subtasks := issue.Fields.Subtasks
	if len(subtasks) > 0 {
		buffer.WriteString("# Subtasks\n")
		for _, s := range subtasks {
			buffer.WriteString(fmt.Sprintf("%s: %s\n", s.Fields.Summary, s.Fields.Description))
		}
	}

	comments := issue.Fields.Comments.Comments
	if len(comments) > 0 {
		buffer.WriteString("# Comments\n")
		for _, c := range comments {
			comment := fmt.Sprintf("**%s**: %s\n", c.Author.DisplayName, c.Body)
			buffer.WriteString(comment)
		}
	}
	out, err := glamour.Render(buffer.String(), "dark")
	if err != nil {
		return fmt.Errorf("couldn't render markdown: %w", err)
	}
	fmt.Println(out)
	return nil
}

func displayIssues(client *Client, issues []string) error {
	for _, i := range issues {
		err := displayIssue(client, i)
		if err != nil {
			return err
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
