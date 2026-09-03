# Contributing

Changes should be small, reviewable and accompanied by tests. Preserve upstream
attribution and do not weaken the default loopback-only listener policy.

Before submitting:

```sh
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

Document user-visible changes in `CHANGELOG.md`. Report vulnerabilities through
the private process in `SECURITY.md`, never through a public issue.
