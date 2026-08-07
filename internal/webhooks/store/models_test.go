package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests defend the sql.Scanner/driver.Valuer contract of JSONMap and
// JSONStringMap. The regression they guard against: JSON columns were once
// mapped to plain Go maps with no Scanner/Valuer, so NULL columns failed to
// scan and maps failed to encode into TEXT/JSONB. The driver may hand Scan a
// nil (NULL), raw bytes, a JSON string, or an already-decoded map (pgx decodes
// jsonb into a Go map), so all four must be handled.

func TestJSONMap_Scan(t *testing.T) {
	tests := []struct {
		name    string
		src     any
		want    JSONMap
		wantErr bool
	}{
		{"nil yields nil map", nil, nil, false},
		{"empty bytes yield nil map", []byte{}, nil, false},
		{"empty string yields nil map", "", nil, false},
		{"json bytes decode", []byte(`{"a":"b","n":1}`), JSONMap{"a": "b", "n": float64(1)}, false},
		{"json string decodes", `{"k":true}`, JSONMap{"k": true}, false},
		{"already-decoded map passes through", map[string]any{"x": float64(2)}, JSONMap{"x": float64(2)}, false},
		{"invalid json errors", []byte(`{not json`), nil, true},
		{"unsupported type errors", 42, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m JSONMap
			err := m.Scan(tt.src)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, m)
		})
	}
}

func TestJSONMap_Value(t *testing.T) {
	t.Run("nil map yields SQL NULL", func(t *testing.T) {
		var m JSONMap
		v, err := m.Value()
		require.NoError(t, err)
		assert.Nil(t, v)
	})

	t.Run("populated map marshals to json bytes", func(t *testing.T) {
		m := JSONMap{"a": "b"}
		v, err := m.Value()
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":"b"}`, string(v.([]byte)))
	})
}

func TestJSONMap_RoundTrip(t *testing.T) {
	orig := JSONMap{"str": "s", "num": float64(3), "nested": map[string]any{"k": "v"}}
	v, err := orig.Value()
	require.NoError(t, err)

	var got JSONMap
	// Value produces []byte; the driver may deliver it back as bytes or string.
	require.NoError(t, got.Scan(v.([]byte)))
	assert.Equal(t, orig, got)
}

func TestJSONStringMap_Scan(t *testing.T) {
	tests := []struct {
		name    string
		src     any
		want    JSONStringMap
		wantErr bool
	}{
		{"nil yields nil map", nil, nil, false},
		{"empty bytes yield nil map", []byte{}, nil, false},
		{"empty string yields nil map", "", nil, false},
		{"json bytes decode", []byte(`{"a":"b"}`), JSONStringMap{"a": "b"}, false},
		{"json string decodes", `{"k":"v"}`, JSONStringMap{"k": "v"}, false},
		{"already-decoded map passes through", map[string]any{"x": "y"}, JSONStringMap{"x": "y"}, false},
		{"non-string value in decoded map errors", map[string]any{"x": float64(1)}, nil, true},
		{"invalid json errors", []byte(`{`), nil, true},
		{"unsupported type errors", 42, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m JSONStringMap
			err := m.Scan(tt.src)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, m)
		})
	}
}

func TestJSONStringMap_Value(t *testing.T) {
	t.Run("nil map yields SQL NULL", func(t *testing.T) {
		var m JSONStringMap
		v, err := m.Value()
		require.NoError(t, err)
		assert.Nil(t, v)
	})

	t.Run("populated map marshals to json bytes", func(t *testing.T) {
		m := JSONStringMap{"h": "v"}
		v, err := m.Value()
		require.NoError(t, err)
		assert.JSONEq(t, `{"h":"v"}`, string(v.([]byte)))
	})
}

func TestJSONStringMap_RoundTrip(t *testing.T) {
	orig := JSONStringMap{"a": "1", "b": "2"}
	v, err := orig.Value()
	require.NoError(t, err)

	var got JSONStringMap
	require.NoError(t, got.Scan(v.([]byte)))
	assert.Equal(t, orig, got)
}

// TestEventRegistration_JSONFieldsRoundTrip exercises the struct-level contract:
// nil JSON fields must round-trip to SQL NULL and back to nil without error,
// which is the exact bootstrap scenario that previously broke push-event.
func TestEventRegistration_NilJSONFieldsAreNullSafe(t *testing.T) {
	reg := EventRegistration{Name: "order.created"}

	// Schema/SamplePayload (JSONMap) and Metadata (JSONStringMap) are nil.
	// A nil JSON field must encode to SQL NULL, not an empty object.
	sv, err := reg.Schema.Value()
	require.NoError(t, err)
	assert.Nil(t, sv)

	spv, err := reg.SamplePayload.Value()
	require.NoError(t, err)
	assert.Nil(t, spv)

	mv, err := reg.Metadata.Value()
	require.NoError(t, err)
	assert.Nil(t, mv)

	// And scanning NULL back yields nil maps (no error) — the exact bootstrap
	// scenario (unset schema/metadata) that previously broke push-event.
	var schema JSONMap
	require.NoError(t, schema.Scan(nil))
	assert.Nil(t, schema)

	var meta JSONStringMap
	require.NoError(t, meta.Scan(nil))
	assert.Nil(t, meta)
}
