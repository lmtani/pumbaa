package workflow

import (
	"encoding/json"
	"errors"
)

// CredentialOptionKeys are the workflow options that carry credentials: a
// service account's private key, an OAuth refresh token, a bearer token.
// They let a run act as whoever submitted it, which is why nothing that
// outlives a submission — the local history, an export — holds them.
var CredentialOptionKeys = []string{"user_service_account_json", "refresh_token", "auth_bearer_token"}

// ErrOptionsNotAnObject reports options that are not a JSON object, and so
// cannot be checked for credentials.
var ErrOptionsNotAnObject = errors.New("workflow options are not a JSON object")

// WithoutCredentials returns the options with every credential key removed,
// and the keys it removed, in CredentialOptionKeys order. Options that hold
// none come back unchanged, byte for byte; options that did are re-encoded,
// keys sorted. An error means the options could not be read as a JSON
// object, so nothing can be said about what they carry.
func WithoutCredentials(options []byte) ([]byte, []string, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(options, &fields); err != nil || fields == nil {
		return nil, nil, ErrOptionsNotAnObject
	}
	var removed []string
	for _, key := range CredentialOptionKeys {
		if _, ok := fields[key]; ok {
			delete(fields, key)
			removed = append(removed, key)
		}
	}
	if len(removed) == 0 {
		return options, nil, nil
	}
	data, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return append(data, '\n'), removed, nil
}
