# Tasks - Version Automation

- [x] Implement Git-based version injection
  - [x] Modify `internal/version/version.go` to use a variable instead of a constant (allowing linker overrides)
  - [x] Update `build/build.sh` to capture `git describe --tags` and inject it via `-ldflags`
  - [x] Update `build/build.bat` to mirror this logic for Windows