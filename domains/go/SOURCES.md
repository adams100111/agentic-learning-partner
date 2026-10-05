# Go Domain Source Map

> This file defines source authority for the Go pack. Existing PyLearn Go documents are candidate references only.

## Current verified runtime baseline

Verified: 2026-10-05.

- Go 1.27 release notes: https://go.dev/doc/go1.27
- Go release history: https://go.dev/doc/devel/release
- Go 1.27 release announcement: https://go.dev/blog/go1.27

Go 1.27.0 was released on 2026-08-19. The release history shows Go 1.27.1 on 2026-09-01.

## Official references by area

- Language specification: https://go.dev/ref/spec
- Effective Go: https://go.dev/doc/effective_go
- Modules reference: https://go.dev/ref/mod
- Memory model: https://go.dev/ref/mem
- Context: https://pkg.go.dev/context
- net/http: https://pkg.go.dev/net/http
- errors: https://pkg.go.dev/errors
- sync: https://pkg.go.dev/sync
- testing: https://pkg.go.dev/testing
- runtime/pprof: https://pkg.go.dev/runtime/pprof
- diagnostics: https://go.dev/doc/diagnostics
- vulnerability management: https://go.dev/doc/security/vuln/

## Go 1.27 features relevant to the pack

The official Go 1.27 release material currently supports teaching, where relevant:

- generic methods;
- promoted-field selectors in struct literals;
- encoding/json/v2;
- the standard-library uuid package;
- goroutine leak profiles;
- current runtime/toolchain changes.

These are version-sensitive and should be rechecked when Go 1.28 becomes the target.

## Ecosystem-source rule

No ecosystem library is selected solely because PyLearn currently uses it.

For libraries such as routers, PostgreSQL drivers, SQL generators, migration tools, validation packages, WebSocket libraries, retry/circuit-breaker libraries, OpenTelemetry integrations, test containers, CLIs, or deployment tooling:

1. identify the competency need first;
2. evaluate current maintained candidates;
3. prefer primary docs/repositories;
4. record `lastVerified`;
5. document why the chosen tool is the current teaching default;
6. teach the underlying Go contract so replacing the library does not invalidate the mental model.

## PyLearn

Useful references:

- `adams100111/pylearn/docs/go/CONSTITUTION.md`
- `adams100111/pylearn/docs/go/CURRICULUM_OVERVIEW.md`

They are input for comparison and platform mapping, not source authority.
