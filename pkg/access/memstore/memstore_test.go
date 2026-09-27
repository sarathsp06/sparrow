package memstore_test

import (
	"testing"

	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/memstore"
	"github.com/sarathsp06/sparrow/pkg/access/storetest"
)

func TestConformance(t *testing.T) {
	storetest.Run(t, func(*testing.T) access.Store { return memstore.New() })
}
