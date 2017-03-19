package message

var (
	MessageAuthenticate = New("C|A+")
	MessageConnected    = New("C|CH+")
	MessageReject       = New("C|REJ+")
	MessagePing         = New("C|PI+")
	MessagePong         = New("C|PO+")
)
