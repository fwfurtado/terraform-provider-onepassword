package client

import "testing"

type mockoonFixture struct {
	VaultID       string
	VaultName     string
	ItemID        string
	ItemTitle     string
	SectionName   string
	UsernameField string
	UsernameValue string
	PasswordField string
	PasswordValue string
}

func newTestClient(t *testing.T) (*ClientWrapper, mockoonFixture) {
	t.Helper()

	server := newMockoonServer(t)
	t.Cleanup(server.Close)

	return newClientWrapper(server.NewClient()), server.Fixture()
}
