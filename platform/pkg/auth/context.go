package auth

import (
	"context"

	"github.com/google/uuid"
)

const SessionTokenKey = "session-uuid"

type (
	userUUIDCtxKey    struct{}
	sessionUUIDCtxKey struct{}
)

// WithUserUUID puts the user UUID into the context.
func WithUserUUID(ctx context.Context, userUUID uuid.UUID) context.Context {
	return context.WithValue(ctx, userUUIDCtxKey{}, userUUID)
}

// UserUUIDFromContext extracts the user UUID from the context.
func UserUUIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userUUID, ok := ctx.Value(userUUIDCtxKey{}).(uuid.UUID)
	return userUUID, ok
}

// WithSessionUUID puts the session UUID into the context.
// It is a pure passthrough from the incoming header/metadata to the outgoing gRPC
// metadata via SessionForwarder, so no parsing is needed here.
func WithSessionUUID(ctx context.Context, sessionUUID uuid.UUID) context.Context {
	return context.WithValue(ctx, sessionUUIDCtxKey{}, sessionUUID)
}

// SessionUUIDFromContext extracts the session UUID from the context.
func SessionUUIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	sessionUUID, ok := ctx.Value(sessionUUIDCtxKey{}).(uuid.UUID)
	return sessionUUID, ok
}
