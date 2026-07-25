# External Dependencies

The module is core-first and offline: it makes **no third-party network calls at
runtime**. All classification data is local (embedded JSON plus any datasets the
deployer loads).

| Name | Version | Project URL | Purpose | Security / privacy notes |
|------|---------|-------------|---------|--------------------------|
| `golang.org/x/net/publicsuffix` | see `go.mod` | https://pkg.go.dev/golang.org/x/net/publicsuffix | Correct eTLD+1 (registrable domain) resolution so multi-part TLDs like `co.uk` are handled right | Maintained by the Go team (the "extended standard library"). No transitive third-party dependencies. Contains a compiled-in snapshot of the Public Suffix List — purely local data, no network access. |

Everything else is the Go standard library.

## Updating the public suffix data

The suffix list is embedded in `golang.org/x/net`. To refresh it, bump the
`x/net` version (`go get golang.org/x/net@latest && go mod tidy`) — this is a
minor-version dependency update, not a runtime fetch.
