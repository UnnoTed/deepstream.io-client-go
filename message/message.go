package message

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/UnnoTed/deepstream.io-client-go/consts"
	"github.com/UnnoTed/deepstream.io-client-go/consts/actions"
	"github.com/UnnoTed/deepstream.io-client-go/consts/events"
	"github.com/UnnoTed/deepstream.io-client-go/consts/topics"
)

type Data struct {
	Bytes []byte
	Text  string
}

func (d *Data) Data() []byte {
	if d.Bytes != nil {
		return d.Bytes
	}

	return []byte(d.Text)
}

func (d *Data) String() string {
	if d.Bytes != nil {
		return string(d.Bytes)
	}

	return d.Text
}

type OnMessage chan *Message
type Message struct {
	Topic  []byte
	Action []byte
	Data   []*Data
	Raw    []byte
}

func String(msg []byte) string {
	m := string(msg)
	m = strings.Replace(m, consts.MessagePartSeparatorString, "|", -1)
	m = strings.Replace(m, consts.MessageSeparatorString, "+", -1)

	return m
}

func New(msg ...interface{}) []byte {
	if len(msg) == 1 {
		return (&Message{}).Build(msg[0].(string))
	}

	var (
		built string
		size  = len(msg) - 1
	)

	for i, part := range msg {
		switch part.(type) {
		case string:
			built += part.(string)

		case topics.Topic:
			built += part.(topics.Topic).String()

		case actions.Action:
			built += part.(actions.Action).String()

		case events.Event:
			built += part.(events.Event).String()

		case map[string]interface{}:
			j, _ := json.Marshal(part.(map[string]interface{}))
			built += string(j)
		}

		if i < size {
			built += consts.MessagePartSeparatorString
		}
	}
	built += consts.MessageSeparatorString

	return []byte(built)
}

func (m *Message) Build(msg string) []byte {
	msg = strings.Replace(msg, "|", consts.MessagePartSeparatorString, -1)
	msg = strings.Replace(msg, "+", consts.MessageSeparatorString, -1)

	return []byte(msg)
}

// Parse the topic, action and message's data from a message (TOPIC|ACTION|[]DATA...)
//                                                           (0    |1     |2|3|4....)
func (m *Message) Parse(data []byte) error {
	m.Data = []*Data{}
	m.Raw = data
	s := bytes.Split(bytes.TrimSuffix(data, consts.MessageSeparators), consts.MessagePartSeparators)

	m.Topic = s[0]
	m.Action = s[1]

	if len(s) > 2 {
		for i := 2; i < len(s); i++ {
			if len(s[i]) == 0 {
				continue
			}

			// data should be string
			if s[i][0] == 'S' {
				m.Data = append(m.Data, &Data{
					Bytes: s[i][1:],
					Text:  string(s[i][1:]),
				})

				// data is []byte
			} else {
				m.Data = append(m.Data, &Data{Bytes: s[i]})
			}
		}
	}

	return nil
}

func (m *Message) String() string {
	msg := string(m.Raw)

	msg = strings.Replace(msg, consts.MessagePartSeparatorString, "|", -1)
	return strings.Replace(msg, consts.MessageSeparatorString, "+", -1)
}

func (m *Message) Equals(m2 []byte) bool {
	return bytes.Equal(m.Raw, m2)
}
