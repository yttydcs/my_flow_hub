package service

import "strings"

// Match checks if a permission node matches a required node, supporting wildcards.
// Example: Match("a.b.c", "a.b.*") -> true
// Example: Match("a.b.c", "a.**") -> true
func Match(required, userNode string) bool {
	if userNode == "**" {
		return true
	}
	if userNode == required {
		return true
	}

	if strings.HasSuffix(userNode, ".**") {
		prefix := strings.TrimSuffix(userNode, ".**")
		if strings.HasPrefix(required, prefix) {
			return true
		}
	}

	if strings.HasSuffix(userNode, ".*") {
		prefix := strings.TrimSuffix(userNode, ".*")
		requiredParts := strings.Split(required, ".")
		prefixParts := strings.Split(prefix, ".")
		if len(requiredParts) == len(prefixParts)+1 && strings.HasPrefix(required, prefix) {
			return true
		}
	}

	return false
}

// CanGrant checks if a user with a set of permissions can grant a specific permission node.
func CanGrant(userPerms map[string]struct{}, nodeToGrant string) bool {
	for p := range userPerms {
		if Match(nodeToGrant, p) {
			return true
		}
	}
	return false
}