package consts

type (
	ConnectionState string
)

var (
	MessageSeparators     = []byte{byte(30)}
	MessagePartSeparators = []byte{byte(31)}
)

const (
	ConnectionStateClosed                 ConnectionState = "CLOSED"
	ConnectionStateAwaitingConnection     ConnectionState = "AWAITING_CONNECTION"
	ConnectionStateChallenging            ConnectionState = "CHALLENGING"
	ConnectionStateAwaitingAuthentication ConnectionState = "AWAITING_AUTHENTICATION"
	ConnectionStateAuthenticating         ConnectionState = "AUTHENTICATING"
	ConnectionStateOpen                   ConnectionState = "OPEN"
	ConnectionStateError                  ConnectionState = "ERROR"
	ConnectionStateReconnecting           ConnectionState = "RECONNECTING"

	MessageSeparatorString     = string(byte(30))
	MessagePartSeparatorString = string(byte(31))
	MessageSeparator           = byte(30)
	MessagePartSeparator       = byte(31)

	TypesString    = "S"
	TypesObject    = "O"
	TypesNumber    = "N"
	TypesNull      = "L"
	TypesTrue      = "T"
	TypesFalse     = "F"
	TypesUndefined = "U"

	CallStateInitial     = "INITIAL"
	CallStateConnecting  = "CONNECTING"
	CallStateEstablished = "ESTABLISHED"
	CallStateAccepted    = "ACCEPTED"
	CallStateDeclined    = "DECLINED"
	CallStateEnded       = "ENDED"
	CallStateError       = "ERROR"
)
