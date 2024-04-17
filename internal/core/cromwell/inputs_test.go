package cromwell

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/lmtani/pumbaa/internal/adapters/logger"

	"github.com/lmtani/pumbaa/internal/adapters/test"

	"github.com/lmtani/pumbaa/internal/types"
)

func TestInputs(t *testing.T) {
	content, err := os.ReadFile("../../adapters/cromwellclient/testdata/metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	meta := types.MetadataResponse{}
	err = json.Unmarshal(content, &meta)
	if err != nil {
		t.Fatal(err)
	}
	fakeCromwell := test.FakeCromwell{
		MetadataResponse: meta,
	}
	l := logger.NewLogger(logger.InfoLevel)
	i := NewCromwell(&fakeCromwell, l)

	inputs, err := i.Inputs("fake-operation")
	if err != nil {
		t.Error(err)
	}

	expected := map[string]interface{}{
		"HelloHere.someFile":  "gs://just-testing/file.txt",
		"HelloHere.someInput": "just testing string",
	}
	if !reflect.DeepEqual(inputs, expected) {
		t.Errorf("Expected %v, got %v", expected, i)
	}

	if !fakeCromwell.MetadataCalled {
		t.Errorf("Expected MetadataCalled to be true, found %v", fakeCromwell.MetadataCalled)
	}
}
