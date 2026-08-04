package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AssertGRPCStatus checks that err is a gRPC error carrying the expected code.
func AssertGRPCStatus(t *testing.T, err error, expectedCode codes.Code) {
	t.Helper()

	require.Error(t, err, "expected an error")

	st, ok := status.FromError(err)
	assert.True(t, ok, "the error must be a gRPC status")
	assert.Equal(t, expectedCode, st.Code(), "unexpected gRPC code: %s", st.Message())
}
