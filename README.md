# GoULC (Go Utils Library Collection) 🚀

[![Go Report Card](https://goreportcard.com/badge/gitlab.com/iglou.eu/goulc)](https://goreportcard.com/report/gitlab.com/iglou.eu/goulc)
[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

GoULC (pronounced as one word) is a collection of Go libraries I developed for my professional and personal usage. Born out of the desire to reduce cascading and/or redundant dependencies across projects, GoULC focuses on lightweight implementations while leveraging Go's standard library whenever possible. While primarily designed for my own use cases, feel free to use it if it fits your needs! 😊

## 📌 Why GoULC?

1. **🔗 Dependency Management**: Tired of importing numerous libraries for basic functionality.
2. **📚 Standard Library First**: Leverages Go's standard library whenever possible.
3. **🛠️ Framework Agnostic**: Core functionality works without external dependencies.
4. **🏢 Real-World Usage**: Actually used in professional environments.

## 🎯 Philosophy

- **Minimal Dependencies**: Preference for standard library implementations where possible.
- **Flexible Integration**: Use of interfaces for maximum adaptability.
- **Security-First**: Careful consideration of dependencies and their impact.
- **Build Tags**: External dependencies are isolated using build tags (e.g., `//go:build gorm`).

## 📦 Available Packages for Now

### 📝 Logging

A light and flexible logging package built on top of `log/slog` that supports multiple output handlers, log levels, and framework integrations.

See the [logging package documentation](logging/README.md) for detailed usage and integration examples.

## 🤝 Contributing

While this is primarily a personal toolkit, contributions are welcome if you find something that could be improved! Feel free to submit a Pull Request. 🎉

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📜 License

This project is licensed under the [GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0) - see the [LICENSE](LICENSE) file for details.

## 🛠️ Support

- **Report bugs** by opening an issue
- **Request features** through issues
- **Ask questions** in discussions

---

Made with ❤️ by [Adrien Kara](https://gitlab.com/adrienK)