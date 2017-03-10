package actions

import "bytes"

type Action []byte

var (
	Ping                          Action = []byte("PI")
	Pong                          Action = []byte("PO")
	Ack                           Action = []byte("A")
	Redirect                      Action = []byte("RED")
	Challenge                     Action = []byte("CH")
	ChallengeResponse             Action = []byte("CHR")
	Read                          Action = []byte("R")
	Create                        Action = []byte("C")
	Update                        Action = []byte("U")
	Patch                         Action = []byte("P")
	Delete                        Action = []byte("D")
	Subscribe                     Action = []byte("S")
	Unsubscribe                   Action = []byte("US")
	Has                           Action = []byte("H")
	Snapshot                      Action = []byte("SN")
	Invoke                        Action = []byte("I")
	SubscriptionForPatternFound   Action = []byte("SP")
	SubscriptionForPatternRemoved Action = []byte("SR")
	SubscriptionHasProvider       Action = []byte("SH")
	Listen                        Action = []byte("L")
	Unlisten                      Action = []byte("UL")
	ListenAccept                  Action = []byte("LA")
	ListenReject                  Action = []byte("LR")
	ProviderUpdate                Action = []byte("PU")
	Createorread                  Action = []byte("CR")
	Event                         Action = []byte("EVT")
	Error                         Action = []byte("E")
	Request                       Action = []byte("REQ")
	Response                      Action = []byte("RES")
	Rejection                     Action = []byte("REJ")
	PresenceJoin                  Action = []byte("PNJ")
	PresenceLeave                 Action = []byte("PNL")
	Query                         Action = []byte("Q")
	WriteAcknowledgement          Action = []byte("WA")
)

func (a Action) Bytes() []byte {
	return []byte(a)
}

func (a Action) String() string {
	return string(a)
}

func (a Action) Equals(data []byte) bool {
	return bytes.Equal(a, data)
}
