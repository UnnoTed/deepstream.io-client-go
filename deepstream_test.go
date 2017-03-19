package deepstream

import (
	"strconv"
	"testing"

	"sync"

	"github.com/UnnoTed/deepstream.io-client-go/message"
	"github.com/stretchr/testify/assert"
	dry "github.com/ungerik/go-dry"
)

var (
	clients = []*DeepStream{}
	dss     = []*DeepStream{{}, {}}
	wait    sync.WaitGroup
)

const serverAddress = "localhost:6020"

func init() {
	wait.Add(len(dss))
}

func TestInit(t *testing.T) {
	dry.RandSeedWithTime()

	for _, ds := range dss {
		go func(ds *DeepStream) {
			connect, err := ds.Connect(serverAddress)
			assert.NoError(t, err)
			<-connect.Channels.Listen()

			auth, err := ds.Authenticate(map[string]interface{}{
				"username": dry.RandomHexString(10),
				"password": dry.RandomHEXString(5),
			})
			assert.NoError(t, err)
			<-auth.Channels.Listen()

			wait.Done()
		}(ds)
	}

	wait.Wait()
}

func TestDeepStreamSingleClient(t *testing.T) {
	//wait.Wait()

	// client 2
	client2 := dss[1]
	log.Info("SUBSCRIBE CLIENT 2")
	sub2, _, err := client2.Events.Subscribe("chat")
	assert.NoError(t, err)
	_ = sub2
	log.Info("WAITING SUBSCRIBE CLIENT 2")
	<-sub2.Channels.Listen()
	log.Info("DONE SUBSCRIBE CLIENT 2")

	// client 1
	log.Debug("[TEST]: Connecting...")
	client := dss[0]
	sub, onChat, err := client.Events.Subscribe("chat")
	<-sub.Channels.Listen()
	log.Info("SUBISCRIBEEEED!!")
	assert.NoError(t, err)

	client2.Events.Emit("chat", []byte("hi"))
	hi := <-onChat.Channels.Listen()
	t.Log(hi.Data)
	assert.Equal(t, "hi", hi.Data[1].String())

	log.Info("EXPECTATION ENDED!")

	unsub, err := client.Events.Unsubscribe("chat")
	assert.NoError(t, err)
	<-unsub.Channels.Listen()

	unsub2, err := client2.Events.Unsubscribe("chat")
	assert.NoError(t, err)
	<-unsub2.Channels.Listen()

	//err = client2.Close()
	//assert.NoError(t, err)
	//
	//err = client.Close()
	//assert.NoError(t, err)
	//
	//log.Info("Closed")
}

func TestDeepStream(t *testing.T) {
	return
	var onChatList []*ExpectedMessage

	for i := 0; i < 10; i++ {
		clients = append(clients, &DeepStream{})
		c, err := clients[i].Connect(serverAddress)
		assert.NoError(t, err)
		assert.NotNil(t, clients[i].Connection)
		<-c.Channels.Listen()

		a, err := clients[i].Authenticate(map[string]interface{}{
			"user": "bot" + strconv.FormatInt(int64(i), 10),
		})
		assert.NoError(t, err)
		<-a.Channels.Listen()

		onSub, onChat, err := clients[i].Events.Subscribe("chat")
		assert.NoError(t, err)
		<-onSub.Channels.Listen()

		onChatList = append(onChatList, onChat)
	}

	var (
		next     int
		size     = len(clients) - 1
		waiter   = &sync.WaitGroup{}
		possible = [][]byte{}
	)

	for i, cl := range clients {
		if i < size {
			next = i + 1
		} else {
			next = 0
		}

		hi := []byte("hi" + strconv.FormatInt(int64(i), 10))
		possible = append(possible, hi)
		go cl.Events.Emit("chat", hi)

		waiter.Add(1)
		go func(onChatList []*ExpectedMessage, next int) {
			chat := <-onChatList[next].Channels.Listen()
			assert.Contains(t, possible, chat.Data[1].Bytes)
			waiter.Done()
		}(onChatList, next)
	}

	waiter.Wait()

	for _, cl := range clients {
		err := cl.Close()
		assert.NoError(t, err)
	}
}

func BenchmarkMessageReader(b *testing.B) {
	ds := dss[0]

	msg := &message.Message{}
	msg.Parse(msg.Build("C|CH+"))

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ds.CheckMessage(msg.Raw)
	}
}
