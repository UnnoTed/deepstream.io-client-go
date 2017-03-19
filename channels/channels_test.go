package channels

import (
	"sync"
	"testing"

	"github.com/UnnoTed/deepstream.io-client-go/message"
	"github.com/stretchr/testify/assert"
)

func TestChannels(t *testing.T) {
	chs := New()

	var wg sync.WaitGroup
	var wg2 sync.WaitGroup

	msg := &message.Message{}
	err := msg.Parse(msg.Build("C|CH+"))
	assert.NoError(t, err)

	// loop
	for i := 0; i < 100; i++ {
		wg.Add(1)
		wg2.Add(1)

		go func(chs *Channels, msg *message.Message, t *testing.T) {
			defer wg.Done()
			wg2.Done()

			m := <-chs.Listen()
			assert.True(t, m.Matches(msg))
		}(chs, msg, t)
	}

	wg2.Wait()

	// send
	chs.Send(msg)

	// wait
	wg.Wait()

	assert.NoError(t, chs.Close())
}
