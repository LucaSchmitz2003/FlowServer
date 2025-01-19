package FlowServer

import (
	"net"
	"strings"
)

// anonymizeIP replaces the last octet of an IPv4 address or the last 4 digits of an IPv6 address with 'xxx'.
func anonymizeIP(ip string) string {
	parsedIP := net.ParseIP(ip)
	if parsedIP.To4() != nil {
		// IPv4
		lastIndex := strings.LastIndex(ip, ".")
		if lastIndex != -1 {
			return ip[:lastIndex] + ".xxx"
		}
	} else if parsedIP.To16() != nil {
		// IPv6
		lastIndex := strings.LastIndex(ip, ":")
		if lastIndex != -1 {
			return ip[:lastIndex] + ":xxxx"
		}
	}
	return ip
}
