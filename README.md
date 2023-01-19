# jira

*Atlassian Jira CLI*

# Setup

[Generate a Jira API token](https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/).
See the [`go-jira` authentication documentation](https://pkg.go.dev/github.com/andygrunwald/go-jira@v1.16.0#readme-authentication).

## Environment variables

Example `.envrc`:

```
export JIRA_TOKEN=ABCDEFGHIJKLMNOP1234567
export JIRA_URL=https://mycorp.atlassian.net/
export JIRA_USER=user@mycorp.com
```
