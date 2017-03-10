package topics

import "bytes"

type Topic []byte

var (
	Connection Topic = []byte("C")
	Auth       Topic = []byte("A")
	Error      Topic = []byte("X")
	Event      Topic = []byte("E")
	Record     Topic = []byte("R")
	RPC        Topic = []byte("P")
	Presense   Topic = []byte("U")
	Private    Topic = []byte("PRIVATE/")
)

func (t Topic) Bytes() []byte {
	return []byte(t)
}

func (t Topic) String() string {
	return string(t)
}

func (t Topic) Equals(data []byte) bool {
	return bytes.Equal(t, data)
}
