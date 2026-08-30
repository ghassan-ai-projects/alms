package service

import (
	"fmt"
	"strings"
)

func buildOKFIndexForFiles(files []OKFBundleFile, options OKFExportOptions, summary OKFExportSummary) string {
	var body strings.Builder
	body.WriteString("---\n")
	body.WriteString("okf_version: \"")
	body.WriteString(okfVersion)
	body.WriteString("\"\n")
	body.WriteString("---\n\n")
	body.WriteString("# ALMS Learning Export\n\n")
	body.WriteString("High-confidence ALMS learnings exported as OKF concept documents.\n\n")
	body.WriteString("# Selection\n\n")
	if options.Query != "" {
		fmt.Fprintf(&body, "- Query: `%s`\n", options.Query)
	} else {
		body.WriteString("- Query: not applied\n")
	}
	if options.Type != "" {
		fmt.Fprintf(&body, "- Type: `%s`\n", options.Type)
	}
	if options.Status != "" {
		fmt.Fprintf(&body, "- Status: `%s`\n", options.Status)
	}
	fmt.Fprintf(&body, "- Minimum score: `%.2f`\n", options.MinScore)
	fmt.Fprintf(&body, "- Matched: `%d`\n", summary.Matched)
	fmt.Fprintf(&body, "- Exported: `%d`\n", summary.Exported)
	body.WriteString("\n# Learnings\n\n")
	if len(files) == 0 {
		body.WriteString("No learnings met the export criteria.\n")
		return body.String()
	}
	for _, file := range files {
		title := extractTitleFromOKFDocument(file.Content)
		fmt.Fprintf(&body, "* [%s](%s) - exported ALMS learning\n", title, file.Path)
	}
	return body.String()
}

func extractTitleFromOKFDocument(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "title: ") {
			return strings.Trim(strings.TrimPrefix(line, "title: "), "\"")
		}
	}
	return "Untitled"
}
