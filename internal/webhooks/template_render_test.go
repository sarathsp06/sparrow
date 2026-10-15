package webhooks

import (
	"errors"
	"fmt"
	"testing"

	svcerrors "github.com/sarathsp06/sparrow/internal/errors"
	"github.com/sarathsp06/sparrow/pkg/template"
)

// A busy template engine is a temporary condition, not a template error.
func TestTemplateRenderError(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want svcerrors.Status
	}{
		{fmt.Errorf("x: %w", template.ErrBusy), svcerrors.Unavailable},
		{errors.New("failed to execute template: boom"), svcerrors.InvalidArgument},
	} {
		var se *svcerrors.ServiceError
		if !errors.As(templateRenderError(tc.err), &se) || se.Status != tc.want {
			t.Errorf("templateRenderError(%v) = %v, want status %v", tc.err, se, tc.want)
		}
	}
}
