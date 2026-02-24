package utils

import "strings"

// ShortName extracts the last segment after "/" from an ARN or path.
// Returns the input unchanged if no "/" is found.
func ShortName(arn string) string {
	if parts := strings.Split(arn, "/"); len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return arn
}

// SecondToLast extracts the second-to-last "/" segment from an ARN.
// Useful for target group ARNs where the name sits before the final segment.
// Returns the input unchanged if fewer than 2 segments exist.
func SecondToLast(arn string) string {
	parts := strings.Split(arn, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return arn
}
