package main

import "strings"

func getExtension(formType string) string {
	if formType == "" {
		return ""
	}
	data := strings.Split(formType, "/")
	if len(data) == 2 {
		return data[1]
	}
	return ""
}
