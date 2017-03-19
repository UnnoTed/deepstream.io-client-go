package deepstream

import (
	"bytes"
	"encoding/json"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"sync"

	"github.com/UnnoTed/deepstream.io-client-go/channels"
	"github.com/UnnoTed/deepstream.io-client-go/consts"
	"github.com/UnnoTed/deepstream.io-client-go/consts/actions"
	"github.com/UnnoTed/deepstream.io-client-go/consts/events"
	"github.com/UnnoTed/deepstream.io-client-go/consts/topics"
	"github.com/UnnoTed/deepstream.io-client-go/message"
	"github.com/gorilla/websocket"
)

var (
	log  *zap.Logger
	comu = &sync.RWMutex{}
)

func init() {
	var err error
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	log, err = config.Build()
	if err != nil {
		panic(err)
	}

	log = log.Named("DeepStream")
}

type ExpectedMessage struct {
	*message.Message

	ID          string
	ShouldClose bool
	Matcher     *message.Matcher
	// C           FutureMessage
	Channels *channels.Channels
}

type DeepStream struct {
	Connection *websocket.Conn
	Closed     bool
	Events     *bus
	Records    *Records

	Wait *sync.WaitGroup

	expectedMessages map[string]*ExpectedMessage
	LastMessage      []byte

	state consts.ConnectionState
	url   string
	emu   *sync.RWMutex
	mu    *sync.RWMutex

	expectingState bool
}

// Connect to a deepstream server
func (ds *DeepStream) Connect(url string) (*ExpectedMessage, error) {
	log.Debug("[Connect] Connecting...")
	comu.RLock()
	if ds.Connection != nil {
		comu.RUnlock()

		log.Info("You're already connected.")
		return nil, nil
	}

	comu.RUnlock()

	if ds.Wait == nil {
		ds.Wait = &sync.WaitGroup{}
		ds.Wait.Add(1) // disables reader
	}

	if ds.emu == nil {
		ds.emu = &sync.RWMutex{}
	}

	if ds.mu == nil {
		ds.mu = &sync.RWMutex{}
	}

	if ds.expectedMessages == nil {
		ds.expectedMessages = map[string]*ExpectedMessage{}
	}

	// sets the state to awaiting connection
	ds.SetState(consts.ConnectionStateAwaitingConnection)

	log.Debug("[Connect] Watching for state changes...")
	// watch for state changes from server
	go func(ds *DeepStream) {
		if ds.expectingState {
			return
		}

		// expect: connection state changed
		ex, err := ds.ExpectMessage(topics.Event, actions.Event, events.ConnectionStateChanged)
		if err != nil {
			log.Error("error connection state changed", zap.Error(err))
			return
		}

		defer ds.Unexpect(ex.ID)
		ds.expectingState = true

		c := ex.Channels.Listen()
		for {
			m := <-c
			if len(m.Data) > 0 {
				ds.SetState(consts.ConnectionState(m.Data[len(m.Data)-1].String()))
			}
		}
	}(ds)

	ds.mu.Lock()
	ds.Closed = true
	ds.mu.Unlock()

	ds.url = url

	// insert ws protocol when not found
	if !strings.HasPrefix(url, "ws://") {
		url = "ws://" + url
	}

	// insert deepstream in the end when not found
	if !strings.HasSuffix(url, "/deepstream") {
		url += "/deepstream"
	}

	log.Debug("[Connect] Starting websocket...")

	// connects to the server
	var err error
	comu.Lock()
	if ds.Connection, _, err = websocket.DefaultDialer.Dial(url, nil); err != nil {
		comu.Unlock()
		return nil, err
	}
	comu.Unlock()

	log.Debug("[Connect] Running reader...")
	go ds.reader()

	// on connect
	log.Debug("[Connect] Expecting connection challenge")
	onConnect, err := ds.ExpectMessageOnce(topics.Connection, actions.Challenge)
	if err != nil {
		return nil, err
	}

	go func(onConnect *ExpectedMessage, ds *DeepStream) {
		log.Debug("Waiting on Connect")
		<-onConnect.Channels.Listen()

		log.Debug("on connect OK, closed = false")
		ds.SetState(consts.ConnectionStateOpen)

		ds.mu.Lock()
		ds.Closed = false
		ds.mu.Unlock()

		// enable reader
		ds.Wait.Done()
	}(onConnect, ds)

	// on redirect
	onRedirect, err := ds.ExpectMessage(topics.Connection, actions.Redirect)
	if err != nil {
		return nil, err
	}

	go func(onRedirect *ExpectedMessage, ds *DeepStream) {
		c := onRedirect.Channels.Listen()

		for {
			msg := <-c

			comu.Lock()
			ds.Connection = nil
			comu.Unlock()

			connect, err := ds.Connect(msg.Data[0].String())
			if err != nil {
				log.Error("error while reconnecting", zap.Error(err))
			}

			<-connect.Channels.Listen()
		}
	}(onRedirect, ds)

	log.Debug("Emiting...")

	// send challenge response
	go ds.Emit(message.New(topics.Connection, actions.ChallengeResponse, ds.url))

	// insert events handlers
	if ds.Events == nil {
		ds.Events = newEvent()
		ds.Events.ds = ds
	}

	if ds.Records == nil {
		ds.Records = newRecords()
		ds.Records.ds = ds
	}

	return onConnect, err
}

// Emit sends a message to the ws as text
func (ds *DeepStream) Emit(msg []byte) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	log.Debug("Emit", zap.String("msg", message.String(msg)))
	err := ds.Connection.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		return err
	}

	log.Debug("Emitted", zap.String("msg", message.String(msg)))
	return nil
}

// SetState sets the local state to the given state and emits
// a event informing that the connection's state has changed
func (ds *DeepStream) SetState(state consts.ConnectionState) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	log.Info("Client state changed", zap.String("old", string(ds.state)), zap.String("new", string(state)))
	ds.state = state
}

// State returns the current connection's state
func (ds *DeepStream) State() consts.ConnectionState {
	if ds.Connection == nil {
		return consts.ConnectionStateAwaitingConnection
	}

	// ds.mu.RLock()
	// defer ds.mu.RUnlock()

	return ds.state
}

// Authenticate sends a auth request to the deepstream server
func (ds *DeepStream) Authenticate(data map[string]interface{}) (*ExpectedMessage, error) {
	ds.SetState(consts.ConnectionStateAuthenticating)

	log.Info("Sending Auth")
	j, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	ex, err := ds.ExpectMessageOnce(topics.Auth, actions.Ack)
	if err != nil {
		return nil, err
	}

	go ds.Emit(message.New(topics.Auth, actions.Request, string(j)))

	return ex, nil
}

// Close closes the ws connection
func (ds *DeepStream) Close() error {
	if ds.Connection != nil {
		return nil
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()

	log.Info("Closing connection...")
	if err := ds.Connection.Close(); err != nil {
		return err
	}

	ds.Closed = true
	ds.SetState(consts.ConnectionStateClosed)
	log.Info("Connection closed.")

	return nil
}

// reader keeps a loop running waiting for messages
// after the message is parsed it checks if the message is expected
// if expected: it sends the message through a channel
func (ds *DeepStream) reader() {
	var first = true

	for {
		comu.RLock()
		if ds.Connection == nil {
			comu.RUnlock()
			return
		}
		comu.RUnlock()

		log.Debug("Waiting for next message...")
		var err error
		_, ds.LastMessage, err = ds.Connection.ReadMessage()
		log.Info("MESSAGE FROM SERVER", zap.String("msg", message.String(ds.LastMessage)))

		ds.mu.RLock()
		if ds.Closed && !first {
			ds.mu.RUnlock()
			return
		}
		ds.mu.RUnlock()

		if first {
			first = false
		}

		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				return
			}

			log.Error(err.Error())
			return
		}

		// responds to ping
		if bytes.Equal(ds.LastMessage, message.MessagePing) {
			log.Debug("Sending pong")
			go ds.Emit(message.MessagePong)
			continue

			// connection rejected
		} else if bytes.Equal(ds.LastMessage, message.MessageReject) {
			ds.Connection = nil
			return

			// waits for auth
		} else if bytes.Equal(ds.LastMessage, message.MessageAuthenticate) {
			log.Debug("WAITING AUTH")
		}

		log.Debug("Parsing message...")

		ds.CheckMessage(ds.LastMessage)

		log.Debug("waiting")
		ds.Wait.Wait()
		log.Debug("waited")
	}
}

func (ds *DeepStream) CheckMessage(msg []byte) error {
	// parse message
	messages, err := message.Parse(msg)
	if err != nil {
		log.Error("Error parsing multiple messages", zap.Error(err))
		return err
	}

	for _, m := range messages {
		log.Debug("Checking for expected message",
			zap.String("msg", m.String()))

		ds.emu.RLock()
		// check if each expected message matches the current message
		// when the message is expected only once, the channel is closed
		for id, ex := range ds.expectedMessages {
			if ex.ShouldClose {
				log.Debug("Expected message should be found once", zap.String("msg", ex.Message.String()))
			}

			// compare messages
			if ex.Message != nil && ex.Message.Matches(m) {
				log.Debug("Found expected message",
					zap.String("expected", ex.Message.String()),
					zap.String("current", m.String()))

				// ignores message when matcher doesn't match
				if ex.Matcher != nil && !ex.Matcher.Matches(m) {
					log.Warn("Ignoring message from matcher", zap.String("msg", ex.Message.String()))
					continue
				}

				// sends message through channel
				ds.emu.RUnlock()
				log.Debug("SENDING MESSAGE TO CHANNELS", zap.String("msg", m.String()))
				ex.Channels.Send(m)
				ds.emu.RLock()

				// close channel when expected only once
				if ex.ShouldClose {
					if err := ds.Unexpect(id); err != nil {
						panic(err)
					}
					log.Debug("Ended expectation for message", zap.String("msg", ex.Message.String()))
				}
			} else {
				log.Debug("Message didn't match", zap.String("expected", ex.Message.String()), zap.String("current", m.String()))
			}
		}
		ds.emu.RUnlock()
	}

	return nil
}

func (ds *DeepStream) expectMessage(shouldClose bool, matcher *message.Matcher, topic topics.Topic, action actions.Action, data ...[]byte) (*ExpectedMessage, error) {
	mm := &message.Message{}
	mm.Parse(message.New(topic, action, data))
	id := mm.String()

	ds.emu.RLock()
	if _, ok := ds.expectedMessages[id]; ok {
		defer ds.emu.RUnlock()
		return ds.expectedMessages[id], nil
	}
	ds.emu.RUnlock()

	// create a expected message with a parsed message and a
	// channel to send events when the expected message is found
	exm := &ExpectedMessage{
		ID:          mm.String(),
		ShouldClose: shouldClose,
		Message:     mm,
		Channels:    channels.New(),
		Matcher:     matcher,
	}

	// insert expected message into list
	ds.emu.Lock()
	defer ds.emu.Unlock()

	ds.expectedMessages[exm.ID] = exm
	return exm, nil
}

// ExpectMessage checks every message for the expected one
// when found sends a event through a channel with the message
func (ds *DeepStream) ExpectMessage(topic topics.Topic, action actions.Action, data ...[]byte) (*ExpectedMessage, error) {
	return ds.expectMessage(false, nil, topic, action, data...)
}

// ExpectMessageOnce checks every message for the expected one
// when found sends a event through a channel with the message
// after the first event is sent, the channel is closed automatically
func (ds *DeepStream) ExpectMessageOnce(topic topics.Topic, action actions.Action, data ...[]byte) (*ExpectedMessage, error) {
	return ds.expectMessage(true, nil, topic, action, data...)
}

func (ds *DeepStream) ExpectMessageWithMatcher(matcher *message.Matcher, topic topics.Topic, action actions.Action, data ...[]byte) (*ExpectedMessage, error) {
	return ds.expectMessage(false, matcher, topic, action, data...)
}

func (ds *DeepStream) Unexpect(id string) error {
	ds.expectedMessages[id].Channels.Close()

	//ds.emu.Lock()
	//defer ds.emu.Unlock()
	delete(ds.expectedMessages, id)

	return nil
}
