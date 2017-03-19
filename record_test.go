package deepstream

import (
	"testing"

	"sync"

	"github.com/stretchr/testify/assert"
	dry "github.com/ungerik/go-dry"
)

func TestRecordInit(t *testing.T) {
	var wg sync.WaitGroup
	ds := &DeepStream{}

	connect, err := ds.Connect(serverAddress)
	assert.NoError(t, err)
	<-connect.Channels.Listen()

	auth, err := ds.Authenticate(map[string]interface{}{
		"username": dry.RandomHexString(10),
		"password": dry.RandomHEXString(5),
	})
	assert.NoError(t, err)
	<-auth.Channels.Listen()

	r, err := ds.Records.GetRecord("potus", "")
	assert.NoError(t, err)
	<-r.OnCreateRecord.Channels.Listen()

	wg.Add(1)
	go func() {
		defer wg.Done()
		msg := <-r.OnRecord.Channels.Listen()
		t.Log(msg)
	}()

	r.Set(map[string]interface{}{
		"current": "Donald Trump",
	})
	wg.Add(1)
	wg.Wait()
}
