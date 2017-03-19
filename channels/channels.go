package channels

import (
	"log"
	"sync"

	"github.com/UnnoTed/deepstream.io-client-go/message"
)

type FutureMessage chan *message.Message

func New() *Channels {
	chs := &Channels{
		mu: &sync.RWMutex{},
	}

	return chs
}

type Channels struct {
	List  []FutureMessage
	mu    *sync.RWMutex
	Debug bool
}

func (chs *Channels) Send(msg *message.Message) {
	if chs.Debug {
		log.Println("send", msg.String())
	}

	chs.mu.RLock()
	defer chs.mu.RUnlock()

	for _, ch := range chs.List {
		ch <- msg
	}
}

func (chs *Channels) Listen() FutureMessage {
	ch := make(FutureMessage)

	chs.mu.Lock()
	defer chs.mu.Unlock()
	if chs.Debug {
		log.Println("listen", len(chs.List))
	}

	chs.List = append(chs.List, ch)

	return ch
}

func (chs *Channels) Close() error {
	if chs.Debug {
		log.Println("close")
	}

	chs.mu.RLock()
	defer chs.mu.RUnlock()

	for _, ch := range chs.List {
		close(ch)
	}

	return nil
}
