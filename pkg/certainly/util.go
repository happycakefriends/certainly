package certainly

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func sanitizeIPv6addr(s string) string {
	// Remove brackets from IPv6 addresses, net.ParseCIDR needs this
	re, _ := regexp.Compile(`[\[\]]+`)
	return re.ReplaceAllString(s, "")
}

func SanitizeString(s string) string {
	// URL safe base64 alphabet without padding as defined in ACME
	re, _ := regexp.Compile(`[^A-Za-z\-\_0-9]+`)
	return re.ReplaceAllString(s, "")
}

func CorrectPassword(pw string, hash string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)); err == nil {
		return true
	}
	return false
}

func WildcardDomains(domains []string) []string {
	doms := []string{}
	for _, domain := range domains {
		doms = append(doms, fmt.Sprintf("*.%s", domain))
	}
	return doms
}

func IsManagedApex(name string, domains []string) bool {
	for _, domain := range domains {
		if name == domain {
			return true
		}
	}
	return false
}

func TransformToWildcard(name string) string {
	nameSlice := strings.Split(name, ".")
	if len(nameSlice) < 2 {
		return name
	}
	nameSlice[0] = "*"
	return strings.Join(nameSlice, ".")
}
