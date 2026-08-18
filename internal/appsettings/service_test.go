package appsettings

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockService(t *testing.T) (*pgxmock.PgxPoolIface, *Service, context.Context) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })
	repo := NewRepository(mock)
	svc := NewService(repo)
	return &mock, svc, context.Background()
}

func TestService_GetAll(t *testing.T) {
	mock, svc, ctx := newMockService(t)
	rows := pgxmock.NewRows([]string{"key", "value"}).
		AddRow("store_name", "My Store").
		AddRow("logo_path", "logo.png")
	(*mock).ExpectQuery("SELECT key, value FROM app_settings").WillReturnRows(rows)

	result, err := svc.GetAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, "My Store", result["store_name"])
	assert.Equal(t, "logo.png", result["logo_path"])
}

func TestService_GetAll_Error(t *testing.T) {
	boom := errors.New("db down")
	mock, svc, ctx := newMockService(t)
	(*mock).ExpectQuery("SELECT key, value FROM app_settings").WillReturnError(boom)

	_, err := svc.GetAll(ctx)
	assert.Error(t, err)
}

func TestService_GetMultiple(t *testing.T) {
	mock, svc, ctx := newMockService(t)
	rows := pgxmock.NewRows([]string{"key", "value"}).
		AddRow("store_name", "Test")
	(*mock).ExpectQuery("SELECT key, value FROM app_settings WHERE key IN").WithArgs(pgxmock.AnyArg()).WillReturnRows(rows)

	result, err := svc.GetMultiple(ctx, []string{"store_name"})
	require.NoError(t, err)
	assert.Equal(t, "Test", result["store_name"])
}

func TestService_Upsert_Success(t *testing.T) {
	mock, svc, ctx := newMockService(t)
	(*mock).ExpectBegin()
	(*mock).ExpectExec("INSERT INTO app_settings").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	(*mock).ExpectExec("INSERT INTO app_settings").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	(*mock).ExpectCommit()

	settings := map[string]string{
		"store_name": "New Store",
		"logo_path":  "new-logo.png",
	}
	err := svc.Upsert(ctx, settings)
	assert.NoError(t, err)
}

func TestService_Upsert_EmptyStoreName(t *testing.T) {
	_, svc, ctx := newMockService(t)

	err := svc.Upsert(ctx, map[string]string{"store_name": "  "})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store_name must not be empty")
}

func TestService_Upsert_TrimSpaces(t *testing.T) {
	mock, svc, ctx := newMockService(t)
	(*mock).ExpectBegin()
	(*mock).ExpectExec("INSERT INTO app_settings").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	(*mock).ExpectCommit()

	err := svc.Upsert(ctx, map[string]string{"store_name": "  Trimmed Name  "})
	assert.NoError(t, err)
}

func TestService_SaveLogoPath(t *testing.T) {
	mock, svc, ctx := newMockService(t)
	(*mock).ExpectBegin()
	(*mock).ExpectExec("INSERT INTO app_settings").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	(*mock).ExpectCommit()

	err := svc.SaveLogoPath(ctx, "saved.png")
	assert.NoError(t, err)
}

func TestService_LogoPathForFile(t *testing.T) {
	path := LogoPathForFile(".png")
	assert.Contains(t, path, "uploads/logos/")
	assert.Contains(t, path, "logo.png")

	path2 := LogoPathForFile(".svg")
	assert.Contains(t, path2, "logo.svg")
}

func TestService_ValidateLogoExtension(t *testing.T) {
	tests := []struct {
		ext     string
		wantErr bool
		wantExt string
	}{
		{".png", false, ".png"},
		{".jpg", false, ".jpg"},
		{".jpeg", false, ".jpeg"},
		{".svg", true, ""}, // SVG no longer supported — raster only
		{".PNG", false, ".png"}, // normalised to lowercase
		{".gif", true, ""},
		{".webp", true, ""},
		{"", true, ""},
		{".exe", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			ext, err := ValidateLogoExtension(tt.ext)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantExt, ext)
			}
		})
	}
}
