package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveAdminPassword(t *testing.T) {
	strong := "S7rong-Passphrase!42"

	t.Run("production without any password fails", func(t *testing.T) {
		_, err := resolveAdminPassword("production", "", "")
		assert.Error(t, err)
	})

	t.Run("staging without any password fails", func(t *testing.T) {
		_, err := resolveAdminPassword("staging", "", "")
		assert.Error(t, err)
	})

	t.Run("production with the former in-code default fails", func(t *testing.T) {
		_, err := resolveAdminPassword("production", devAdminPassword, "")
		assert.Error(t, err)
	})

	t.Run("production with a denylisted password fails", func(t *testing.T) {
		_, err := resolveAdminPassword("production", "password123456", "")
		assert.Error(t, err)
	})

	t.Run("production with a short password fails", func(t *testing.T) {
		_, err := resolveAdminPassword("production", "short", "")
		assert.Error(t, err)
	})

	t.Run("production with a strong flag password passes", func(t *testing.T) {
		got, err := resolveAdminPassword("production", strong, "")
		assert.NoError(t, err)
		assert.Equal(t, strong, got)
	})

	t.Run("production reads ADMIN_PASSWORD when the flag is empty", func(t *testing.T) {
		got, err := resolveAdminPassword("production", "", strong)
		assert.NoError(t, err)
		assert.Equal(t, strong, got)
	})

	t.Run("production rejects a weak ADMIN_PASSWORD", func(t *testing.T) {
		_, err := resolveAdminPassword("production", "", devAdminPassword)
		assert.Error(t, err)
	})

	t.Run("flag wins over ADMIN_PASSWORD", func(t *testing.T) {
		got, err := resolveAdminPassword("production", strong, "0ther-Str0ng-Pass!")
		assert.NoError(t, err)
		assert.Equal(t, strong, got)
	})

	t.Run("development without a password falls back to the dev default", func(t *testing.T) {
		got, err := resolveAdminPassword("development", "", "")
		assert.NoError(t, err)
		assert.Equal(t, devAdminPassword, got)
	})

	t.Run("development accepts a weak password without validation", func(t *testing.T) {
		got, err := resolveAdminPassword("development", "abc", "")
		assert.NoError(t, err)
		assert.Equal(t, "abc", got)
	})
}
