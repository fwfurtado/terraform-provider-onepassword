package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientWrapperMockoonFixture(t *testing.T) {
	client, fixture := newTestClient(t)

	require.NotNil(t, client)
	require.NotEmpty(t, fixture.VaultID)
	require.NotEmpty(t, fixture.ItemID)
}
