package workflow

import (
	"errors"
	"strings"
	"testing"
)

func TestWithoutCredentials(t *testing.T) {
	data, removed, err := WithoutCredentials([]byte(`{"refresh_token": "1//x", "jes_gcs_root": "gs://b", "auth_bearer_token": "t", "user_service_account_json": "{}"}`))
	if err != nil {
		t.Fatalf("WithoutCredentials: %v", err)
	}
	if strings.Join(removed, ",") != strings.Join(CredentialOptionKeys, ",") {
		t.Errorf("removed = %v, want every credential key in CredentialOptionKeys order", removed)
	}
	if string(data) != "{\n  \"jes_gcs_root\": \"gs://b\"\n}\n" {
		t.Errorf("data = %q, want only the other options", data)
	}

	clean := []byte(`{"read_from_cache":false}`)
	data, removed, err = WithoutCredentials(clean)
	if err != nil || removed != nil || string(data) != string(clean) {
		t.Errorf("clean options = %q %v %v, want them back byte for byte", data, removed, err)
	}

	for _, bad := range []string{"", "null", "[]", "not json", `"string"`} {
		if _, _, err := WithoutCredentials([]byte(bad)); !errors.Is(err, ErrOptionsNotAnObject) {
			t.Errorf("WithoutCredentials(%q) err = %v, want ErrOptionsNotAnObject", bad, err)
		}
	}
}
