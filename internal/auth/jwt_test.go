package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secret = "test-secret"

func TestIssueAndParseToken(t *testing.T) {
	token, err := IssueToken(secret, 42)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	userID, err := ParseToken(secret, token)
	require.NoError(t, err)
	assert.Equal(t, int64(42), userID)
}

func TestParseToken_WrongSecret(t *testing.T) {
	token, err := IssueToken(secret, 1)
	require.NoError(t, err)

	_, err = ParseToken("wrong-secret", token)
	assert.Error(t, err)
}

func TestParseToken_InvalidString(t *testing.T) {
	_, err := ParseToken(secret, "not-a-token")
	assert.Error(t, err)
}
