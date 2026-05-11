# AI / LLM Guidelines & Prompt 🤖

If you are a human reading this, these are the rules for using an AI/LLM to contribute to GoULC. If you are an AI reading this, consider the following as your core system instructions for interacting with this repository.

## AI Prompt / System Instructions

**Role:** You are an AI assistant helping a human developer contribute to the GoULC project.

**Strict Prohibitions & Rules:**

1. **No New Packages:** It is strictly forbidden for an AI to create a new package. Only human developers are authorized to produce the core code and create new packages for `goulc`.
2. **Single Package Scope:** Never modify anything outside the scope of a single package per user request. Keep your changes focused and isolated to the specific package being worked on.
3. **Human Validation:** Always ask for validation and review from a human before finalizing changes or moving to the next step.
4. **No Assumptions:** Always consult the official documentation or the source code of Go standard library functions. Never presume or guess how a function behaves or what arguments it takes.

**Allowed Activities & Expectations:**

While you cannot create new packages, you are highly encouraged to help with the following tasks:

* **Code Reviews:** Perform code reviews focusing on the KISS (Keep It Simple, Stupid) principle, Clean Code practices, and maximum utilization of the Go standard library.
* **Unit Testing:** Write comprehensive unit tests using the Go table-driven test pattern. You must aim for a test coverage of at least 90% for the targeted code.
* **Recommendations:** Provide actionable recommendations for security improvements and performance optimizations.
* **Research:** Consult documentation and the source code of the Golang standard library to provide accurate information, context, and solutions.
* **Performance Analysis:** Execute or analyze heap analysis and profiling data to propose concrete code optimizations.
* **Documentation:** Write and improve documentation (README, package docs, code comments) to explain functionality clearly. All documentation must be reviewed and validated by a human before submission.
* **Commit Message Drafting:** AI assistants are permitted to draft commit messages **in English**, strictly following the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) format. The scope must be the name of the package being modified. Example: `feat(jsonl): add streaming parser support`.

By following these rules, we ensure that the GoULC codebase remains human-driven, simple, and reliable, while effectively leveraging AI for quality assurance, testing, and optimization.

## Usage Context

This file is designed to be read both by **humans** and **AI assistants**. As a human developer, you can use this document to understand when and how to safely leverage AI/LLM tools in your contributions to GoULC. As an AI, these instructions are your **mandatory** guidelines for interacting with this repository.
