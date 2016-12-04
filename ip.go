package netx

import "net"

// IsIP checks if s is a valid representation of an IPv4 or IPv6 address.
func IsIP(s string) bool {
	return net.ParseIP(s) != nil
}

// IsIPv4 checks if s is a valid representation of an IPv4 address.
func IsIPv4(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() != nil
}

// IsIPv6 checks if s is a valid representation of an IPv6 address.
func IsIPv6(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() == nil
}
