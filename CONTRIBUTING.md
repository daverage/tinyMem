# Contributing to tinyMem

Thank you for your interest in contributing to tinyMem! As an evidence-based memory system, we have strict guidelines to ensure truth discipline and reliability.

## Principles

1.  **Truth Discipline**: Never auto-promote model output to truth. Facts require local evidence verification.
2.  **Streaming First**: All LLM interactions must remain streaming-first to ensure low latency for the user.
3.  **Local Execution**: No cloud dependencies. All state must remain project-scoped in `.tinyMem/`.
4.  **Zero Config**: Sensible defaults must work out of the box.

## Development Workflow

1.  **Fork and Clone**: Create your feature branch from `main`.
2.  **Go Modules**: We use Go 1.22+. Ensure your `go.mod` and `go.sum` are updated.
3.  **Tests**: Run standard tests with `go test ./...`. Run the full Python test suite in `test/` before submitting a PR.
4.  **Doctor**: If you add new functionality that can fail, update `internal/doctor` to include a diagnostic check.

## Pull Request Guidelines

-   Ensure all tests pass.
-   Update `README.md` or relevant documentation in `docs/` if you change behavior.
-   Keep PRs focused and atomic.

## License

By contributing to tinyMem, you agree that your contributions will be licensed under the project's MIT License.
