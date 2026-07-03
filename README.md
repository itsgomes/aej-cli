# AEJ

AEJ is a small command-line interface for Jira. It lets you inspect issues and boards, search for work, and manage worklogs directly from your terminal.

## Tech Stack

| Technology | Purpose |
|---|---|
| [Go](https://go.dev) 1.24+ | Application language |
| [Cobra](https://github.com/spf13/cobra) | CLI framework |
| [Viper](https://github.com/spf13/viper) | Configuration management |
| [Resty](https://github.com/go-resty/resty) | HTTP client |

## Running the Project

Clone the repository and download its dependencies:

```bash
git clone https://github.com/itsgomes/aej-cli.git
cd aej-cli
go mod download
```

Run it directly:

```bash
go run ./cmd/aej --help
```

Or build a local executable:

```bash
go build -o aej ./cmd/aej       # Linux / macOS
go build -o aej.exe ./cmd/aej   # Windows
```

To install it in Go's standard binary directory (`GOBIN`, or `$GOPATH/bin` by default):

```bash
go install ./cmd/aej
```

To use a dedicated `$GOPATH/aej` directory instead, set `GOBIN` to that absolute path before running `go install`, then add the directory to your `PATH`.

## Configuration

Configure your Jira Cloud URL, Atlassian email, and API token interactively:

```bash
aej login
```

For automation and CI, use environment variables instead:

```bash
export AEJ_JIRA_URL="https://your-company.atlassian.net"
export AEJ_EMAIL="you@company.com"
export AEJ_API_TOKEN="your-token"
```

Run `aej --help` to see all available commands.

## Notes

- AEJ targets the Jira Cloud REST API v3. Jira Server and Data Center may require endpoint changes.
- `aej board` requires Jira Software and access to the board APIs.
- The Jira URL must use HTTPS and contain only the scheme and host.
- Terminal colors are disabled when output is redirected and when `NO_COLOR` is set.

## Development

```bash
gofmt -w .
go vet ./...
go test ./...
```
