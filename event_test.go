package deepstream

import (
	"testing"

	"regexp"

	"sync"

	"github.com/UnnoTed/deepstream.io-client-go/message"
	"github.com/gobwas/glob"
	"github.com/stretchr/testify/assert"
)

func TestEventListenerOffline(t *testing.T) {
	ds := dss[0]
	var wg sync.WaitGroup
	wg.Add(1)

	expected := &message.Message{}
	expected.Parse(expected.Build("E|EVT|911/truth|jet fuel can't melt steel beams+"))

	// ds
	matcher, err := message.NewMatcher(glob.MustCompile("911/*"))
	assert.NoError(t, err)

	onListen, onMessage, err := ds.Events.Listen(matcher, "^911/.*")
	assert.NoError(t, err)
	<-onListen.Channels.Listen()
	log.Info("LISTENED")

	go func(onMessage *ExpectedMessage) {
		defer wg.Done()
		log.Info("WAITING MSG")

		msg := <-onMessage.Channels.Listen()
		assert.Equal(t, expected.String(), msg.String())
		log.Info("DONE")
	}(onMessage)

	log.Info("SENT")
	go ds.CheckMessage(expected.Raw)
	wg.Wait()
	log.Info("WAITED")
}

func TestEventSubscription(t *testing.T) {
	var wg sync.WaitGroup
	ds := dss[0]
	ds2 := dss[1]

	// ds
	onSub, onChat, err := ds.Events.Subscribe("chat")
	assert.NoError(t, err)
	<-onSub.Channels.Listen()

	// ds2
	onSub2, onChat2, err := ds2.Events.Subscribe("chat")
	assert.NoError(t, err)
	<-onSub2.Channels.Listen()

	// ds -> ds2
	planes := &message.Message{}
	planes.Parse(planes.Build("E|EVT|chat|planes+"))
	wg.Add(1)

	go func(onChat2 *ExpectedMessage, planes *message.Message) {
		defer wg.Done()
		m := <-onChat2.Channels.Listen()
		assert.True(t, m.Matches(planes))
	}(onChat2, planes)

	go ds.Events.Emit("chat", []byte("planes"))
	wg.Wait()
	return

	// ds2 -> ds
	buildings := &message.Message{}
	buildings.Parse(buildings.Build("E|EVT|chat|buildings+"))
	wg.Add(1)

	go func(onChat *ExpectedMessage, buildings *message.Message) {
		defer wg.Done()
		m := <-onChat.Channels.Listen()
		assert.True(t, m.Matches(buildings))
	}(onChat, buildings)

	go ds2.Events.Emit("chat", []byte("buildings"))
	wg.Wait()

	// ds
	onUnsub, err := ds.Events.Unsubscribe("chat")
	assert.NoError(t, err)
	<-onUnsub.Channels.Listen()

	// ds2
	onUnsub2, err := ds2.Events.Unsubscribe("chat")
	assert.NoError(t, err)
	<-onUnsub2.Channels.Listen()

	// ds2
	go func() {
		<-onChat2.Channels.Listen()
		assert.Fail(t, "ds2 is still subscribed")
	}()

	// ds
	go ds.Events.Emit("chat", []byte("planes"))
}

func TestEventListener(t *testing.T) {
	return
	log.Warn("LISTENER")
	var wg sync.WaitGroup
	wg.Add(2)

	ds := dss[0]
	ds2 := dss[1]

	expected := &message.Message{}
	expected.Parse(expected.Build("E|EVT|911/truth|jet fuel can't melt steel beams+"))

	// ds
	matcher, err := message.NewMatcher(glob.MustCompile("911/*"))
	assert.NoError(t, err)
	assert.True(t, matcher.Matches(expected))

	onListen, onMessage, err := ds.Events.Listen(matcher, "^911/.*")
	assert.NoError(t, err)
	<-onListen.Channels.Listen()
	log.Debug("listened")

	var msg1, msg2 = &message.Message{}, &message.Message{}
	go ds2.Events.Emit("911/truth", []byte("jet fuel can't melt steel beams123"))

	go func(onMessage *ExpectedMessage) {
		defer wg.Done()
		msg1 = <-onMessage.Channels.Listen()
	}(onMessage)

	// ds2
	matcher2, err := message.NewMatcher(regexp.MustCompile(`^911/.*`))
	assert.NoError(t, err)
	assert.True(t, matcher2.Matches(expected))

	onListen2, onMessage2, err := ds2.Events.Listen(matcher2, "^911/.*")
	assert.NoError(t, err)
	<-onListen2.Channels.Listen()

	go func(onMessage2 *ExpectedMessage) {
		defer wg.Done()
		msg2 = <-onMessage2.Channels.Listen()
	}(onMessage2)

	go ds.Events.Emit("911/truth", []byte("jet fuel can't melt steel beams321"))
	go ds2.Events.Emit("911/truth", []byte("jet fuel can't melt steel beams123"))

	wg.Wait()

	assert.Equal(t, expected, msg1.String())
	assert.Equal(t, expected, msg2.String())
}
