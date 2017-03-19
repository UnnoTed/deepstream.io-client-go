package deepstream

import (
	"sync"

	"github.com/UnnoTed/deepstream.io-client-go/consts/actions"
	"github.com/UnnoTed/deepstream.io-client-go/consts/topics"
	"github.com/UnnoTed/deepstream.io-client-go/message"
)

type Record struct {
	Name string
	Data *message.Data

	HasProvider bool
	IsDestroyed bool
	IsReady     bool
	Usages      int64
	Version     int64

	OnCreateRecord *ExpectedMessage
	OnRecord       *ExpectedMessage

	ds *DeepStream
}

func (r *Record) Delete() {}
func (r *Record) Emit()   {}
func (r *Record) Get() (*ExpectedMessage, error) {
	go r.ds.Emit(message.New(topics.Record, actions.Createorread, r.Name))

	return nil, nil
}

// Set updates the record's data
func (r *Record) Set(data ...interface{}) {
	r.Version++

	// R|U|name|version|data
	go r.ds.Emit(message.New(topics.Record, actions.Update, r.Name, r.Version, data))
}

func (r *Record) On()           {}
func (r *Record) Once()         {}
func (r *Record) Subscribe()    {}
func (r *Record) Unsubscribe()  {}
func (r *Record) WhenReady()    {}
func (r *Record) HasListeners() {}
func (r *Record) Listeners()    {}

type Records struct {
	ds *DeepStream
	mu *sync.RWMutex
}

func newRecords() *Records {
	r := &Records{}

	return r
}

func (rcs *Records) GetAnonymousRecord() error {
	return nil
}

func (rcs *Records) GetList(name, options string) error {
	return nil
}

func (rcs *Records) GetRecord(name, options string) (*Record, error) {
	var (
		err error
		r   = &Record{
			Name: name,
			ds:   rcs.ds,
		}
	)

	// R|A|S|name
	r.OnCreateRecord, err = rcs.ds.ExpectMessageOnce(topics.Record, actions.Ack, actions.Subscribe, []byte(name))
	if err != nil {
		return nil, err
	}

	go func(r *Record) {
		msg := <-r.OnCreateRecord.Channels.Listen()
		r.Version
	}(r)

	// emit: R|CR|name
	go rcs.ds.Emit(message.New(topics.Record, actions.Createorread, name))

	// R|R|name -> R|R|name|version|data
	r.OnRecord, err = rcs.ds.ExpectMessage(topics.Record, actions.Read, []byte(name))
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (rcs *Records) Listen(matcher *message.Matcher, pattern string) error {
	return nil
}

func (rcs *Records) Snapshot(name string) error {
	return nil
}

func (rcs *Records) Has(name string) {
}

func (rcs *Records) Unlisten(pattern string) error {
	return nil
}

/*
getAnonymousRecord: function ()
getList: function ( name, options )
getRecord: function ( name, recordOptions )
has: function ( name, callback )
listen: function ( pattern, callback )
snapshot: function ( name, callback )
unlisten: function ( pattern )
*/
