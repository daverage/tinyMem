# Tasks – Package for GitHub Packages

- [ ] Decide package type and target registry (blocked: need package type from user)
  - [ ] Choose between Container (ghcr.io), npm, Maven, NuGet, RubyGems, etc. (blocked)
  - [ ] Confirm scope/namespace (user or org) (blocked)
- [ ] Set up publish configuration (blocked: depends on package type)
  - [ ] Add registry config files or metadata as needed (blocked)
  - [ ] Configure versioning and package name (blocked)
- [ ] Add GitHub Actions workflow for publish (blocked: depends on package type)
  - [ ] Configure permissions and authentication (blocked)
  - [ ] Set triggers (tag/release) (blocked)
- [ ] Verify publish steps and document usage (blocked: depends on package type)
