package events

import "bytes"

type Event []byte

var (
	ConnectionError                 Event = []byte("connectionError")
	ConnectionStateChanged          Event = []byte("connectionStateChanged")
	MaxReconnectionAttemptsReached  Event = []byte("MAX_RECONNECTION_ATTEMPTS_REACHED")
	ConnectionAuthenticationTimeout Event = []byte("CONNECTION_AUTHENTICATION_TIMEOUT")
	AckTimeout                      Event = []byte("ACK_TIMEOUT")
	NoRPCProvider                   Event = []byte("NO_RPC_PROVIDER")
	ResponseTimeout                 Event = []byte("RESPONSE_TIMEOUT")
	DeleteTimeout                   Event = []byte("DELETE_TIMEOUT")
	UnsolicitedMessage              Event = []byte("UNSOLICITED_MESSAGE")
	MessageDenied                   Event = []byte("MESSAGE_DENIED")
	MessageParseError               Event = []byte("MESSAGE_PARSE_ERROR")
	VersionExists                   Event = []byte("VERSION_EXISTS")
	NotAuthenticated                Event = []byte("NOT_AUTHENTICATED")
	MessagePermissionError          Event = []byte("MESSAGE_PERMISSION_ERROR")
	ListenerExists                  Event = []byte("LISTENER_EXISTS")
	NotListening                    Event = []byte("NOT_LISTENING")
	TooManyAuthAttempts             Event = []byte("TOO_MANY_AUTH_ATTEMPTS")
	IsClosed                        Event = []byte("IS_CLOSED")
	RecordNotFound                  Event = []byte("RECORD_NOT_FOUND")
	NotSubscribed                   Event = []byte("NOT_SUBSCRIBED")
)

func (e Event) Bytes() []byte {
	return []byte(e)
}

func (e Event) String() string {
	return string(e)
}

func (e Event) Equals(data []byte) bool {
	return bytes.Equal(e, data)
}
