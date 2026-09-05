<div align = "center">
  <h1>armature-apprise</h1>

  Apprise HTTP notification client and helpers for jwr SOA apps.

  [![Maturity Badge](https://img.shields.io/badge/maturity-alpha-red.svg)](#none)

</div>

armature-apprise sends JSON payloads to [Apprise](https://github.com/caronc/apprise) `/notify/{key}` endpoints with retries, URL validation, tag helpers, and `{{placeholder}}` message templates.

Apps keep domain-specific notification copy, cvars, and delivery audit tables. This module handles the Apprise HTTP contract only.

## Packages

| Package | Role |
|---------|------|
| `client` | `Payload`, `Notify`, persistent `/notify/{key}` URL validation |
| `tag` | Per-person Apprise tag strings and OR expressions |
| `template` | `{{placeholder}}` rendering for notification bodies |

## Usage

```go
cfg := client.DefaultConfig(client.FormatUserAgent("StarApp", buildinfo.Version))
err := client.Notify(cfg, httpClient, appriseURL, client.Payload{
    Title: "Chore completed",
    Body:  body,
    Tag:   tag.Person("starapp_uid_", memberID),
})
```

Validate admin-configured URLs before save:

```go
if err := client.ValidateNotifyURL(rawURL); err != nil { ... }
```

Develop with a sibling checkout:

```
Development/jamesread/armature-apprise
Development/StarApp
```

`go.mod` in the app uses `replace github.com/jamesread/armature-apprise => ../armature-apprise` (adjust path to your layout).

## Development

```bash
make test
make gocyclo
make golangci
```
