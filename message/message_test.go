package message

import (
	"fmt"
	"strings"
	"testing"

	"github.com/UnnoTed/deepstream.io-client-go/consts"
	"github.com/UnnoTed/deepstream.io-client-go/consts/actions"
	"github.com/UnnoTed/deepstream.io-client-go/consts/topics"
	"github.com/stretchr/testify/assert"
)

var (
	strlist = []string{
		"C|CH+",
		"E|S|chat+",
		"E|A|S|chat+",
		"E|EVT|chat+",
		"E|EVT|chat|hi+",
	}

	bytelist = [][]interface{}{
		{topics.Event, actions.Ack, actions.Subscribe, []byte("chat")},
	}

	matchlist = []struct {
		m1    string
		m2    []interface{}
		match bool
	}{
		{"C|A+", []interface{}{topics.Connection, actions.Ack}, true},
		{"A|A+", []interface{}{topics.Auth, actions.Ack}, true},
		{"E|EVT|chat+", []interface{}{topics.Event, actions.Event, []byte("chat"), []byte("hi")}, true},
		{"E|EVT|connectionStateChanged+", []interface{}{topics.Event, actions.Event, []byte("chat"), []byte("hi")}, false},
	}
)

func getMultipleMessages() []byte {
	var msg []byte

	for _, str := range strlist {
		str = strings.Replace(str, "|", consts.MessagePartSeparatorString, -1)
		str = strings.Replace(str, "+", consts.MessageSeparatorString, -1)

		msg = append(msg, []byte(str)...)
	}

	return msg
}

func TestParseMultiple(t *testing.T) {
	msg := getMultipleMessages()
	list, err := Parse(msg)
	assert.NoError(t, err)
	assert.Equal(t, len(strlist), len(list))

	for i, m := range list {
		// log.Println(m.String(), strlist[i], m, i)
		assert.Equal(t, strlist[i], m.String())
	}
}

func TestParse(t *testing.T) {
	m := &Message{}

	for _, msg := range strlist {
		err := m.Parse(m.Build(msg))
		assert.NoError(t, err)

		assert.Equal(t, msg, m.String())
		t.Log(String(m.Build(msg)), m.String())
	}

	for _, msg := range bytelist {
		err := m.Parse(New(msg...))
		assert.NoError(t, err)

		assert.Equal(t, String(New(msg...)), m.String())
		t.Log(String(New(msg...)), m.String())
	}
}

func TestMatch(t *testing.T) {
	m1, m2 := &Message{}, &Message{}

	for _, msg := range matchlist {
		err := m1.Parse(m1.Build(msg.m1))
		assert.NoError(t, err)

		err = m2.Parse(New(msg.m2...))
		assert.NoError(t, err)

		l := map[string]interface{}{
			"m1":    m1.String(),
			"m2":    m2.String(),
			"match": m1.Matches(m2),
		}

		if msg.match {
			assert.True(t, m1.Matches(m2), fmt.Sprintf("%v", l))

		} else {
			assert.False(t, m1.Matches(m2), fmt.Sprintf("%v", l))
		}
	}
}

func BenchmarkParse(b *testing.B) {
	m := &Message{}
	msg := m.Build(strlist[len(strlist)-1])

	for n := 0; n < b.N; n++ {
		m.Parse(msg)
	}
}

func BenchmarkParseMultiple(b *testing.B) {
	msg := getMultipleMessages()

	for n := 0; n < b.N; n++ {
		Parse(msg)
	}
}
