package message

import "github.com/UnnoTed/deepstream.io-client-go/consts"

const (
	psep = consts.MessagePartSeparator
	sep  = consts.MessageSeparator
)

var (
	// MessageConnected is the message sent when a client connects: C|CH+
	//MessageAuthenticate = bm("A", psep, "REQ", psep, `null`, sep)
	MessageAuthenticate = New("C|A+")
	MessageConnected    = New("C|CH+")
	MessagePing         = New("C|PI+")
	MessagePong         = New("C|PO+")

	messagePrivate = New("PRIVATE/|CH+")
)

func bm(stuff ...interface{}) []byte {
	var data []byte

	for _, arg := range stuff {
		switch arg.(type) {
		case string:
			data = append(data, []byte(arg.(string))...)

		case rune:
			data = append(data, []byte(string(arg.(rune)))...)

		case byte:
			data = append(data, arg.(byte))

		case []byte:
			data = append(data, arg.([]byte)...)
		}
	}

	return data
}
