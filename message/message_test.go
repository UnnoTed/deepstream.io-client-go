package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	m := &Message{}

	err := m.Parse(MessageConnected)
	assert.NoError(t, err)

	t.Log(m)
}

func BenchmarkParseNonPrivate(b *testing.B) {

	for n := 0; n < b.N; n++ {
		m := &Message{}
		m.Parse(MessageConnected)
	}
}

func BenchmarkParsePrivate(b *testing.B) {

	for n := 0; n < b.N; n++ {
		m := &Message{}
		m.Parse(messagePrivate)
	}
}
