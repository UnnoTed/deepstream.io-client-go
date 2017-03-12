package deepstream

import (
	"strconv"
	"testing"

	"sync"

	"github.com/stretchr/testify/assert"
	dry "github.com/ungerik/go-dry"
)

var (
	clients = []*DeepStream{}
)

const serverAddress = "localhost:6020"

func TestDeepStreamSingleClient(t *testing.T) {
	// client 2
	client2 := &DeepStream{}
	connect2, err := client2.Connect(serverAddress)
	assert.NoError(t, err)
	assert.NotNil(t, client2.Connection)
	<-connect2

	log.Info("WAITING FOR AUTH")
	auth2, err := client2.Authenticate(map[string]interface{}{
		"username": dry.RandomHEXString(10),
		"password": "okkk",
	})
	assert.NoError(t, err)
	<-auth2
	log.Info("AUTHENTICATED")

	log.Info("SUBSCRIBE CLIENT 2")
	sub2, _, err := client2.Events.Subscribe("chat")
	assert.NoError(t, err)
	_ = sub2
	log.Info("WAITING SUBSCRIBE CLIENT 2")
	<-sub2
	log.Info("DONE SUBSCRIBE CLIENT 2")

	// client 1
	log.Debug("[TEST]: Connecting...")
	client := &DeepStream{}
	connect, err := client.Connect(serverAddress)
	assert.NoError(t, err)
	assert.NotNil(t, client.Connection)
	assert.NotNil(t, connect)
	<-connect
	log.Debug("[TEST]: Connected")

	auth, err := client.Authenticate(map[string]interface{}{
		"username": dry.RandomHEXString(10),
		"password": "hi",
	})
	assert.NoError(t, err)
	<-auth
	log.Info("AUTHED")

	sub, onChat, err := client.Events.Subscribe("chat")
	<-sub
	log.Info("SUBISCRIBEEEED!!")
	assert.NoError(t, err)

	client2.Events.Emit("chat", []byte("hi"))
	hi := <-onChat
	t.Log(hi.Data)
	assert.Equal(t, "hi", hi.Data[1].String())

	log.Info("EXPECTATION ENDED!")

	err = client2.Close()
	assert.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)

	log.Info("Closed")
}

func TestDeepStream(t *testing.T) {
	onChatList := []FutureMessage{}

	for i := 0; i < 10; i++ {
		clients = append(clients, &DeepStream{})
		c, err := clients[i].Connect(serverAddress)
		assert.NoError(t, err)
		assert.NotNil(t, clients[i].Connection)
		<-c

		a, err := clients[i].Authenticate(map[string]interface{}{
			"user": "bot" + strconv.FormatInt(int64(i), 10),
		})
		assert.NoError(t, err)
		<-a

		onSub, onChat, err := clients[i].Events.Subscribe("chat")
		assert.NoError(t, err)
		<-onSub

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
		go func(onChatList []FutureMessage, next int) {
			chat := <-onChatList[next]
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
