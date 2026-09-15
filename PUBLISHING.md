# Publishing releases

The public module path is `github.com/Bigous/iLBC-Go`. The package import path is `github.com/Bigous/iLBC-Go/src`. The Go package name is
`ilbc`. Version 1 uses tags such as `v1.0.0` without a /v1 module suffix.

## Validate and release

From the repository root in PowerShell:

```powershell
.\tools\coverage.ps1
go vet ./...
go -C cmd/talk test ./...
go -C cmd/talk vet ./...
go test ./src -run '^$' -fuzz FuzzDecode -fuzztime 60s -parallel 4
git diff --check
```

Keep 100% statement coverage, review reference comparisons, and wait for the
GitHub codec, cross-platform talk, race, and fuzz jobs to pass on the release commit. Preserve LICENSE,
LICENSE-RFC3951, NOTICE, and reference source notices when distributing.

Commit and push the release changes, then create a tag on that tested commit:

```powershell
git tag -a v1.0.0 -m 'Release v1.0.0'
git push origin v1.0.0
gh release create v1.0.0 --verify-tag --title 'v1.0.0' --notes-file release-notes.md
```

Use a prepared release-notes file describing the public API and validation.
For subsequent releases, substitute a new semantic version. Never move or
reuse a published version tag. Breaking API changes after v1 require a new
major version and the corresponding module-path suffix.

## Request Go indexing

After publishing the tag, request the version from the public Go proxy:

```powershell
$env:GOPROXY = 'https://proxy.golang.org'
go mod download -json github.com/Bigous/iLBC-Go@v1.0.0
```

This publishes the module to the Go proxy cache and discovery feed; there is
no separate upload operation for index.golang.org. The feed is chronological,
so use a recent RFC3339 `since` timestamp when looking for a new release.
Module paths containing uppercase letters use escaped forms in proxy URLs:
`github.com/!bigous/i!l!b!c-!go`.

Visit https://pkg.go.dev/github.com/Bigous/iLBC-Go@v1.0.0/src.
If the page is not available yet, use its Request button. Processing and search
visibility may lag behind proxy availability. Documentation display also
depends on pkg.go.dev's license detection; retain accurate third-party terms
even if an automated detector does not recognize them.

Official instructions:
- https://pkg.go.dev/about#adding-a-package
- https://proxy.golang.org/
- https://pkg.go.dev/license-policy
