package tools

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	"strings"
)

/**
 * Get the User Id conext from the Metadata
 */
func GetUIDFromContext(ctx context.Context) (string, error) {
	// retrieve incoming metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		// get the first (and presumably only) user ID from the request metadata
		userID := md.Get("User")
		return userID[0], nil
	}
	return "", nil
}


func GetIsAdminStatusFromContext(ctx context.Context) (bool, error) {
	// retrieve incoming metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		// get the first (and presumably only) user ID from the request metadata
		IsAdmin := md.Get("IsAdmin")
		return strings.EqualFold(IsAdmin[0], "true"), nil
	}
	return false, nil
}
