package test_helpers

import (
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
