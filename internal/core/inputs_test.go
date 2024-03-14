package core

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/lmtani/pumbaa/internal/adapters"
	"github.com/lmtani/pumbaa/internal/types"
)

func TestInputs(t *testing.T) {
	content, err := os.ReadFile("../adapters/testdata/metadata.json")
	if err != nil {
		t.Fatal(err)
	}
	meta := types.MetadataResponse{}
	err = json.Unmarshal(content, &meta)
	if err != nil {
		t.Fatal(err)
	}
	fakeCromwell := adapters.FakeCromwell{
		MetadataResponse: meta,
	}

	i := NewInputs(&fakeCromwell)

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
