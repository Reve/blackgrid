package ipam

import (
	"fmt"
	"net/netip"
)

// ValidateCIDR checks if a given string is a valid CIDR notation.
func ValidateCIDR(cidr string) error {
	_, err := netip.ParsePrefix(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR prefix %q: %w", cidr, err)
	}
	return nil
}

// ValidateIP checks if a given string is a valid IP address.
func ValidateIP(ip string) error {
	_, err := netip.ParseAddr(ip)
	if err != nil {
		return fmt.Errorf("invalid IP address %q: %w", ip, err)
	}
	return nil
}

// GetNextAvailableIP returns the first available IP address in the given CIDR prefix
// that is not in the list of existing IPs.
// For IPv4, it skips the network address (all zeros) and broadcast address (all ones).
func GetNextAvailableIP(cidr string, existingIPs []string) (string, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid CIDR prefix %q: %w", cidr, err)
	}
    prefix = prefix.Masked() // Ensure it's the network prefix

	existingSet := make(map[netip.Addr]bool)
	for _, ipStr := range existingIPs {
		if ip, err := netip.ParseAddr(ipStr); err == nil {
			existingSet[ip] = true
		}
	}

	// Calculate broadcast address for IPv4
	var broadcast netip.Addr
	if prefix.Addr().Is4() && prefix.Bits() < 31 {
		b := prefix.Addr().As4()
		maskLen := prefix.Bits()
		hostBits := 32 - maskLen
		for i := 0; i < hostBits; i++ {
			byteIdx := 3 - (i / 8)
			bitIdx := i % 8
			b[byteIdx] |= (1 << bitIdx)
		}
		broadcast = netip.AddrFrom4(b)
	}

	// Start iterating from the first IP
	currentIP := prefix.Addr()

    // Skip network address if it's an IPv4 subnet smaller than /31
    if prefix.Addr().Is4() && prefix.Bits() < 31 {
        currentIP = currentIP.Next()
    }

	for prefix.Contains(currentIP) {
        // Skip broadcast address
		if currentIP.Compare(broadcast) == 0 {
			break
		}

		if !existingSet[currentIP] {
			return currentIP.String(), nil
		}
		currentIP = currentIP.Next()
	}

	return "", fmt.Errorf("no available IP addresses in prefix %s", cidr)
}
