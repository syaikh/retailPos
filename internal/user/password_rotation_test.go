package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createRotationUser creates an account stamped with must_change_password and
// returns it (username unique per test).
func createRotationUser(t *testing.T, username string, mustChange bool) *User {
	t.Helper()
	user := &User{
		Username:           username,
		Email:              username + "@test.com",
		Password:           testPasswordHash(),
		RoleID:             1,
		IsActive:           true,
		MustChangePassword: mustChange,
	}
	require.NoError(t, NewRepository(dbPool).CreateUser(context.Background(), user))
	return user
}

func TestCreateUser_StampsMustChangePassword(t *testing.T) {
	repo := NewRepository(dbPool)
	ctx := context.Background()

	createRotationUser(t, "rotation_stamped", true)
	createRotationUser(t, "rotation_plain", false)

	got, err := repo.GetByUsername(ctx, "rotation_stamped")
	require.NoError(t, err)
	assert.True(t, got.MustChangePassword, "the wizard's temp-password account must be stamped")

	gotPlain, err := repo.GetByUsername(ctx, "rotation_plain")
	require.NoError(t, err)
	assert.False(t, gotPlain.MustChangePassword, "an account created without the flag must not be forced")
}

func TestLogin_CarriesMustChangePasswordClaim(t *testing.T) {
	svc := newAuthServiceWithDB(t)
	ctx := context.Background()

	createRotationUser(t, "rotation_claim", true)

	resp, err := svc.Login(ctx, "rotation_claim", "password")
	require.NoError(t, err)
	assert.True(t, resp.User.MustChangePassword, "the session payload must expose the flag to the client")

	claims, err := svc.ValidateToken(resp.AccessToken)
	require.NoError(t, err)
	assert.True(t, claims.MustChangePassword, "the access token must carry the flag so middleware can enforce 428")
}

func TestChangePassword_ClearsFlagAndReissuesTokens(t *testing.T) {
	svc := newAuthServiceWithDB(t)
	ctx := context.Background()

	createRotationUser(t, "rotation_change", true)

	resp, err := svc.Login(ctx, "rotation_change", "password")
	require.NoError(t, err)
	oldRefresh := resp.RefreshToken

	newAccess, newRefresh, err := svc.ChangePassword(ctx, mustUserID(t, "rotation_change"), "password", "rotatedpass123")
	require.NoError(t, err)
	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)

	claims, err := svc.ValidateToken(newAccess)
	require.NoError(t, err)
	assert.False(t, claims.MustChangePassword, "the re-issued access token must let the caller continue")

	// The flag must be cleared in the database, not only in the new token.
	got, err := NewRepository(dbPool).GetByUsername(ctx, "rotation_change")
	require.NoError(t, err)
	assert.False(t, got.MustChangePassword)

	// The old refresh token is dead, the re-issued one keeps the session alive.
	_, _, _, err = svc.RefreshToken(ctx, oldRefresh)
	assert.Error(t, err, "the pre-change refresh token must be invalidated")

	_, refreshedRefresh, _, err := svc.RefreshToken(ctx, newRefresh)
	require.NoError(t, err, "the re-issued refresh token must keep the session alive")
	assert.NotEmpty(t, refreshedRefresh)
}

func mustUserID(t *testing.T, username string) int {
	t.Helper()
	u, err := NewRepository(dbPool).GetByUsername(context.Background(), username)
	require.NoError(t, err)
	return u.ID
}
