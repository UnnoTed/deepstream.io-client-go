package message

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/UnnoTed/deepstream.io-client-go/consts"
	"github.com/UnnoTed/deepstream.io-client-go/consts/actions"
	"github.com/UnnoTed/deepstream.io-client-go/consts/events"
	"github.com/UnnoTed/deepstream.io-client-go/consts/topics"
)

// Data stores message's data as []byte or string
type Data struct {
	Bytes []byte
	Text  string
}

// Data returns data's []byte or text field as string
func (d *Data) Data() []byte {
	if d.Bytes != nil {
		return d.Bytes
	}

	return []byte(d.Text)
}

// String returns data's []byte as string or text field
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

// String returns the message as string using | and + instead of Ascii control characters separators
func String(msg []byte) string {
	m := string(msg)
	m = strings.Replace(m, consts.MessagePartSeparatorString, "|", -1)
	m = strings.Replace(m, consts.MessageSeparatorString, "+", -1)

	return m
}

// New takes topics, actions, events, strings, map[string]interface{}, [][]byte (from args) or []byte... to build a message
func New(msg ...interface{}) []byte {
	if len(msg) == 1 && reflect.TypeOf(msg[0]).Kind() == reflect.String {
		return (&Message{}).Build(msg[0].(string))
	}

	var (
		built string
		size  = len(msg) - 1
	)

	for i, part := range msg {
		switch part.(type) {
		default:
			built += fmt.Sprintf("%v", part)
		case []interface{}:
			built += string(New(part.([]interface{})...))
		case [][]byte:
			psize := len(part.([][]byte)) - 1

			// insert each bytes slice into the message string
			// also adds | between each item of the slice
			for j, p := range part.([][]byte) {
				built += string(p)

				if j < psize {
					built += consts.MessagePartSeparatorString
				}
			}

			// remove last | if it's the last char in the string
			if built[len(built)-1] == consts.MessagePartSeparator {
				built = built[:len(built)-1]
			}
		case []byte:
			built += string(part.([]byte))
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

// Build replaces | and + with the Ascii control characters separators
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

// String replaces the ansii separators from the message with | and +
// and return the message as string
func (m *Message) String() string {
	return String(m.Raw)
}

func (m *Message) StringData() string {
	msg := ""
	total := len(m.Data) - 1
	for i, data := range m.Data {
		msg += data.String()

		if i < total {
			msg += consts.MessagePartSeparatorString
		}
	}

	msg += consts.MessageSeparatorString

	// m := string(msg)
	// m = strings.Replace(m, consts.MessagePartSeparatorString, "|", -1)
	// m = strings.Replace(m, consts.MessageSeparatorString, "+", -1)

	return msg
}

// Equals checks if a message equals another
func (m *Message) Equals(m2 *Message) bool {
	return bytes.Equal(m.Raw, m2.Raw)
}

// Matches checks if a message is part of a expected message
// E|EVT|chat|hi+ matches expected: E|EVT|chat
func (m *Message) Matches(m2 *Message) bool {
	if bytes.Equal(m.Topic, m2.Topic) && bytes.Equal(m.Action, m2.Action) {
		if (len(m.Data) > 0 && len(m.Data) <= len(m2.Data)) ||
			(len(m.Data) == len(m2.Data)) ||
			(len(m.Data) == 0 && len(m2.Data) > 0) {

			for i := range m.Data {
				if !bytes.Equal(m.Data[i].Bytes, m2.Data[i].Bytes) {
					return false
				}
			}

			return true
		}
	}

	return false
}

// Parse parses multiple messages in a single byte slice
// sample string: E|EVT|chat+E|EVT|chat|hi+E|EVT|chat|hello+
func Parse(data []byte) ([]*Message, error) {
	var (
		list []*Message
		err  error
	)

	// remove last "+" and splits the rest of the "+"
	messages := bytes.Split(bytes.TrimSuffix(data, consts.MessageSeparators), consts.MessageSeparators)

	// insert missing "+", parse the message and insert into the list
	for _, msg := range messages {
		m := &Message{}

		if !bytes.HasSuffix(msg, consts.MessageSeparators) {
			msg = append(msg, consts.MessageSeparators...)
		}

		if err = m.Parse(msg); err != nil {
			return nil, err
		}

		list = append(list, m)
	}

	return list, nil
}
