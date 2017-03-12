package message

var (
	MessageAuthenticate = New("C|A+")
	MessageConnected    = New("C|CH+")
	MessagePing         = New("C|PI+")
	MessagePong         = New("C|PO+")
)
