package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsNullOrEmptyJSON(t *testing.T) {
	t.Parallel()

	require.True(t, isNullOrEmptyJSON(nil))
	require.True(t, isNullOrEmptyJSON([]byte{}))
	require.True(t, isNullOrEmptyJSON([]byte("null")))
	require.False(t, isNullOrEmptyJSON([]byte("{}")))
}

func TestDecodeJSONAnyMap(t *testing.T) {
	t.Parallel()

	t.Run("nil and null values return nil map", func(t *testing.T) {
		t.Parallel()

		decodedNil, err := decodeJSONAnyMap(nil)
		require.NoError(t, err)
		require.Nil(t, decodedNil)

		decodedNull, err := decodeJSONAnyMap([]byte("null"))
		require.NoError(t, err)
		require.Nil(t, decodedNull)
	})

	t.Run("valid json object decodes", func(t *testing.T) {
		t.Parallel()

		decoded, err := decodeJSONAnyMap([]byte(`{"name":"sparrow","retries":3}`))
		require.NoError(t, err)
		require.Equal(t, map[string]any{
			"name":    "sparrow",
			"retries": float64(3),
		}, decoded)
	})

	t.Run("malformed json returns error", func(t *testing.T) {
		t.Parallel()

		decoded, err := decodeJSONAnyMap([]byte(`{"name":`))
		require.Error(t, err)
		require.Nil(t, decoded)
	})
}

func TestDecodeJSONStringMap(t *testing.T) {
	t.Parallel()

	t.Run("nil and null values return nil map", func(t *testing.T) {
		t.Parallel()

		decodedNil, err := decodeJSONStringMap(nil)
		require.NoError(t, err)
		require.Nil(t, decodedNil)

		decodedNull, err := decodeJSONStringMap([]byte("null"))
		require.NoError(t, err)
		require.Nil(t, decodedNull)
	})

	t.Run("empty object decodes", func(t *testing.T) {
		t.Parallel()

		decoded, err := decodeJSONStringMap([]byte(`{}`))
		require.NoError(t, err)
		require.Equal(t, map[string]string{}, decoded)
	})

	t.Run("valid string map decodes", func(t *testing.T) {
		t.Parallel()

		decoded, err := decodeJSONStringMap([]byte(`{"x-test":"true","x-env":"ci"}`))
		require.NoError(t, err)
		require.Equal(t, map[string]string{
			"x-test": "true",
			"x-env":  "ci",
		}, decoded)
	})

	t.Run("malformed json returns error", func(t *testing.T) {
		t.Parallel()

		decoded, err := decodeJSONStringMap([]byte(`{"x-test":`))
		require.Error(t, err)
		require.Nil(t, decoded)
	})
}
