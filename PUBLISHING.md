# Preparing for publication

The local repository is at `C:\dev\projects\iLBC-Go`.

Before the first publication:

1. Run the test suite and review the outstanding items in `VALIDATION.md`:

   ```powershell
   go test ./... '-coverprofile=coverage.out'
   go tool cover '-func=coverage.out'
   go test -run '^$' -bench . -benchmem
   go test -fuzz FuzzDecode -fuzztime 60s
   ```

2. Confirm 100% coverage and update `README.md` and `VALIDATION.md` with
   the actual results. This target has not yet been verified.
3. Choose the public module path, such as `github.com/YOUR_USERNAME/iLBC-Go`,
   and update the `module` directive in `go.mod`, the import in
   `example_test.go`, and the installation examples in the README.
4. Preserve the reference attribution in `LICENSE` and
   `testdata/reference/`. Define the project's distribution terms with
   the included reference source in mind.
5. Configure the chosen remote and publish once the tests pass.
   No remote or release was configured during this preparation.

The workflow in `.github/workflows/ci.yml` runs tests and static analysis
on GitHub with Go 1.22 and the stable release. It has not yet been executed.
