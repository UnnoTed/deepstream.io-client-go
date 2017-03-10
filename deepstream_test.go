package deepstream

import (
	"testing"

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
		"username": "kev2" + dry.RandomHEXString(10),
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
		"username": "kev" + dry.RandomHEXString(10),
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
}

func TestDeepStream(t *testing.T) {
	return
	// for i := 0; i < 100; i++ {
	// 	clients = append(clients, &DeepStream{})
	// 	err := clients[i].Connect("localhost:6020")
	// 	assert.NoError(t, err)
	// 	assert.NotNil(t, clients[i].Connection)

	// 	err = clients[i].Authenticate(map[string]interface{}{
	// 		"user": "kev" + strconv.FormatInt(int64(i), 10),
	// 	})
	// 	assert.NoError(t, err)

	// 	clients[i].Events.Subscribe("chat")
	// 	go clients[i].Events.Emit("chat", []byte("hi"))
	// }

	// time.Sleep(5 * time.Second)

	// for _, cl := range clients {
	// 	err := cl.Close()
	// 	assert.NoError(t, err)
	// }
}
