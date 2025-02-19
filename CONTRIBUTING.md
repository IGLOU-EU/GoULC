# Contributing 😊

Thank you for your interest in contributing to GoULC! We welcome contributions from everyone, and we believe kindness and collaboration make our project great. 💖

## Code Licensing 🔒

- All contributions must be made under the GNU General Public License v3.0 (GPLv3).

## Commit Guidelines 📝

- Adhere to the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) format for all commit messages.
- All commits need to be signed.

## Requirements for Pull Requests ✅

Before submitting a pull request, please ensure that:

- The new code is covered with appropriate unit tests (when relevant).
- There is a package-level README page explaining the functionality of new features or updates.
- The main README is updated with an entry for new package addition.
- No external libraries are used, except standard library or those already part of GoULC.
- For functionality that requires external libraries, conditional builds (e.g., using `//go:build`) must be used to isolate extra dependencies.
- An example demonstrating the usage of the new feature is included in the `example` folder.

## Code Standards 👔

Use tools or libraries only when necessary, and keep dependencies to a minimum. Contributions should reflect this philosophy:

- Keep implementations simple and focused on solving real-world problems.
- Rely on the standard library where possible.
- Ensure that new features are documented and tested properly for smooth integration into workflows.

## How to Contribute 🚀

1. Fork the repository.
2. Create a feature branch (e.g., `git checkout -b feature/your-feature-name`).
3. Commit your changes following the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) guidelines. All commits need to be signed.
4. Push to your branch (`git push origin feature/your-feature-name`).
5. Open a pull request and detail your changes and the reasoning behind them.

## Need Help? 🤔

If you're unsure about anything or have questions on how to get started, please open an issue or reach out on our Matrix channel.

Thank you for helping make GoULC better for everyone! 🎉
