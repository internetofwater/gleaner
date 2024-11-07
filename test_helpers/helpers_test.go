package test_helpers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssertLinesMatchDisregardingOrder(
	t *testing.T,
) {

	expected := "hello\nworld\n"
	actual := "world\nhello\n"
	res := AssertLinesMatchDisregardingOrder(expected, actual)
	assert.True(t, res)

	expected = "123456789\n123456789\n"
	actual = "hello\nworld\n"
	res = AssertLinesMatchDisregardingOrder(expected, actual)
	assert.False(t, res)

	expected = "123456789\n123456789\n123"
	actual = "123456789\n123456789\n"
	res = AssertLinesMatchDisregardingOrder(expected, actual)
	assert.False(t, res)
}

func TestServeStaticFile(t *testing.T) {

	server, listener, err := ServerForStaticFile()
	assert.NoError(t, err)
	defer func() {
		server.Close()
		listener.Close()
	}()
	resp, err := http.Get("http://" + listener.Addr().String() + "/small_sitemap.xml")
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}
