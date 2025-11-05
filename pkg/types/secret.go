package types

import (
	"database/sql/driver"
	"fmt"

	"github.com/pkg/errors"
)

var (
	ErrEmptySecret = errors.New("types.Secret: is empty")
)

type Secret struct {
	value string
}

func NewSecret(value string) (*Secret, error) {
	if value == "" {
		return nil, ErrEmptySecret
	}
	// Perform password validation logic
	return &Secret{value: value}, nil
}

func (p *Secret) Set(value string) error {
	secret, err := NewSecret(value)
	if err != nil {
		return err
	}
	*p = *secret
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
// The secret value is always marshaled to "******" string.
func (p *Secret) MarshalJSON() ([]byte, error) {
	return []byte(`"******"`), nil
}

// String implements the fmt.Stringer interface
// The secret value is always returned as "******" string.
func (p Secret) String() string {
	return "******"
}

// Secret returns the secret value as plain text.
func (p *Secret) Secret() string {
	if p == nil {
		return ""
	}
	return p.value
}

func (p Secret) Value() (driver.Value, error) {
	return p.value, nil
}

// Value implements the driver.Valuer interface.
var _ driver.Valuer = (*Secret)(nil)

// Scan implements the sql.Scanner interface.
// This function scans a value into a Secret pointer.
func (p *Secret) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil
	case string:
		if src == "" {
			return nil
		}
		// Create a new Secret from the string
		secret, err := NewSecret(src)
		if err != nil {
			return errors.Wrapf(err, "types.Secret: unable to scan string(%s) into Secret", src)
		}

		// Assign the new Secret to the pointer
		*p = *secret
	case []byte:
		if len(src) == 0 {
			return nil
		}
		// Create a new Secret from the byte slice
		secret, err := NewSecret(string(src))
		if err != nil {
			return errors.Wrapf(err, "types.Secret: unable to scan []bytes(%s) into Secret", src)
		}
		// Assign the new Secret to the pointer
		*p = *secret
	default:
		return fmt.Errorf("types.Secret: unable to scan type %T into Secret", src)
	}
	return nil
}
