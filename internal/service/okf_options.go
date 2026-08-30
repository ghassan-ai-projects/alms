package service

import "strings"

func normalizeOKFOptions(options OKFExportOptions) OKFExportOptions {
	options.Query = strings.TrimSpace(options.Query)
	options.Type = strings.TrimSpace(options.Type)
	options.Status = strings.TrimSpace(options.Status)
	if options.Status == "" {
		options.Status = defaultOKFStatus
	}
	if options.Status == "all" {
		options.Status = ""
	}
	if options.Limit <= 0 {
		options.Limit = defaultOKFLimit
	}
	if options.MinScore <= 0 {
		options.MinScore = defaultOKFMinScore
	}
	return options
}
