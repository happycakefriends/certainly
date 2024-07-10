package util

import (
	"fmt"
	"strings"
)

func ReplaceApex(target string, rewrites map[string]string) string {
	// Rewrites map is not dot terminated
	tmptarget := target
	if strings.HasSuffix(target, ".") {
		tmptarget = target[:len(target)-1]
	}

	for from, to := range rewrites {
		if strings.HasSuffix(tmptarget, fmt.Sprintf(".%s", from)) || tmptarget == from {
			return replaceLast(target, from, to)
		}
	}
	return target
}

func replaceLast(data, from, to string) string {
	i := strings.LastIndex(data, from)
	if i == -1 {
		return data
	}
	return data[:i] + to + data[i+len(from):]
}

func ShouldRewrite(target string, rewrites map[string]string) bool {
	// Rewrites map is not dot terminated
	if strings.HasSuffix(target, ".") {
		target = target[:len(target)-1]
	}

	for from, _ := range rewrites {
		if strings.HasSuffix(target, fmt.Sprintf(".%s", from)) || target == from {
			return true
		}
	}
	return false
}

func HasApexDomain(target, apex string) bool {
	target = strings.TrimSuffix(target, ".")
	return strings.HasSuffix(strings.ToLower(target), strings.ToLower(apex))
}
