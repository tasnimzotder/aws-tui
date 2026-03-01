package vpc

// NormalizeProtocol converts AWS numeric protocol strings to human-readable names.
func NormalizeProtocol(protocol string) string {
	switch protocol {
	case "-1":
		return "All"
	case "6":
		return "TCP"
	case "17":
		return "UDP"
	case "1":
		return "ICMP"
	default:
		return protocol
	}
}
