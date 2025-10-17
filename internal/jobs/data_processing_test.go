package jobs

import "testing"

func TestDataProcessingArgsKind(t *testing.T) {
	args := DataProcessingArgs{
		DataID:   123,
		DataType: "test",
	}

	if args.Kind() != "data_processing" {
		t.Errorf("Expected Kind() to return 'data_processing', got '%s'", args.Kind())
	}
}
