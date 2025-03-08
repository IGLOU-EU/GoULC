// Package utils provides http utility functions.
package utils

// PathFormatting format the path to remove superfluous / or add / at beginning
func PathFormatting(p string) string {
	if p == "" || p == "/" {
		return "/"
	}

	if p[0] != '/' {
		p = "/" + p
	}

	if p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}

	return p
}
