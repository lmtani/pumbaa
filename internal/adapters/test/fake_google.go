package test

// FakeGoogleCloud is a fake implementation of GoogleCloudPlatform
type FakeGoogleCloud struct{}

// GetStorageClient returns a fake storage client
func (f *FakeGoogleCloud) GetStorageClient() (interface{}, error) {
	return nil, nil
}

// GetIAPToken returns a fake IAP token
func (f *FakeGoogleCloud) GetIAPToken() (string, error) {
	return "fake_iap_token", nil
}

// NewFakeGoogleCloud returns a new FakeGoogleCloud
func NewFakeGoogleCloud() *FakeGoogleCloud {
	return &FakeGoogleCloud{}
}
