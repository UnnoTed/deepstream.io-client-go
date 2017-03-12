package deepstream

import (
	"bytes"
	"encoding/json"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"sync"

	"github.com/UnnoTed/deepstream.io-client-go/consts"
	"github.com/UnnoTed/deepstream.io-client-go/consts/actions"
	"github.com/UnnoTed/deepstream.io-client-go/consts/events"
	"github.com/UnnoTed/deepstream.io-client-go/consts/topics"
	"github.com/UnnoTed/deepstream.io-client-go/message"
	"github.com/gorilla/websocket"
)

var log *zap.Logger

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

type DeepStream struct {
	Connection *websocket.Conn
	Closed     bool
	Events     *bus

	expectedMessages []ExpectedMessage
	LastMessage      []byte
	lmc              chan []byte

	state consts.ConnectionState
	url   string
	emu   *sync.RWMutex
	mu    *sync.RWMutex

	expectingState bool
}

// Connect to a deepstream server
func (ds *DeepStream) Connect(url string) (FutureMessage, error) {
	if ds.Connection != nil {
		log.Info("You're already connected.")
		return nil, nil
	}

	ds.emu = &sync.RWMutex{}
	ds.mu = &sync.RWMutex{}

	// sets the state to awaiting connection
	ds.SetState(consts.ConnectionStateAwaitingConnection)

	// watch for state changes
	go func(ds *DeepStream) {
		if ds.expectingState {
			return
		}

		// expect: connection state changed
		ex := ds.ExpectMessage(topics.Event, actions.Event, events.ConnectionStateChanged)
		defer close(ex)

		ds.expectingState = true

		for {
			m := <-ex
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

	// connects to the server
	var err error
	if ds.Connection, _, err = websocket.DefaultDialer.Dial(url, nil); err != nil {
		return nil, err
	}

	ds.lmc = make(chan []byte)
	go ds.reader()

	log.Debug("Expecting message once")
	ex := ds.ExpectMessageOnce(topics.Connection, actions.Challenge)

	onConnect := ds.ExpectMessageOnce(topics.Connection, actions.Challenge)
	go func(onConnect FutureMessage, ds *DeepStream) {
		log.Debug("Waiting on Connect")
		<-onConnect
		log.Debug("on connect OK, closed = false")

		ds.SetState(consts.ConnectionStateOpen)

		ds.mu.Lock()
		ds.Closed = false
		ds.mu.Unlock()
	}(onConnect, ds)

	log.Debug("Emiting...")

	// send challenge response
	go ds.Emit(message.New(topics.Connection, actions.ChallengeResponse, ds.url))

	ds.Events = newEvent()
	// ds.Events.onUnsubscribe = ds.onUnsubscribe
	ds.Events.onSubscribe = ds.onSubscribe
	ds.Events.onEmit = ds.onEmit

	return ex, err
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
func (ds *DeepStream) Authenticate(data map[string]interface{}) (FutureMessage, error) {
	ds.SetState(consts.ConnectionStateAuthenticating)

	log.Info("Sending Auth")
	j, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	ex := ds.ExpectMessageOnce(topics.Auth, actions.Ack)
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

func (ds *DeepStream) reader() {
	var (
		toRemove []int
		first    = true
	)

	for {
		if ds.Connection == nil {
			return
		}

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
		toRemove = nil
		first = false

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

			// waits for auth
		} else if bytes.Equal(ds.LastMessage, message.MessageAuthenticate) {
			log.Debug("WAITING AUTH")
		}

		log.Debug("Parsing message...")

		// parse message
		messages, err := message.Parse(ds.LastMessage)
		if err != nil {
			log.Error("Error parsing multiple messages", zap.Error(err))
		}

		for _, m := range messages {
			log.Debug("Checking for expected message",
				zap.String("msg", m.String()))

			ds.emu.RLock()
			// check if each expected message matches the current message
			// when the message is expected only once, the channel is closed
			for i, ex := range ds.expectedMessages {
				if ex.ShouldClose {
					log.Debug("Expected message should be found once", zap.String("msg", ex.Message.String()))
				}

				// compare messages
				if ex.Message.Matches(m) {
					log.Debug("Found expected message",
						zap.String("expected", ex.Message.String()),
						zap.String("current", m.String()))

					// sends message through channel
					ds.emu.RUnlock()
					ex.Channel <- m
					ds.emu.RLock()

					// close channel when expected only once
					if ex.ShouldClose {
						close(ex.Channel)
						log.Debug("Ended expectation for message", zap.String("msg", ex.Message.String()))
						toRemove = append(toRemove, i)
					}
				} else {
					log.Debug("Message didn't match", zap.String("expected", ex.Message.String()), zap.String("current", m.String()))
				}
			}
			ds.emu.RUnlock()

			ds.emu.Lock()
			// remove messages expected only once
			if len(toRemove) > 0 {
				offset := 0
				for _, i := range toRemove {
					ds.expectedMessages = append(ds.expectedMessages[:i-offset], ds.expectedMessages[i+1-offset:]...)
					offset++
				}
			}
			ds.emu.Unlock()
		}
	}
}

func (ds *DeepStream) onUnsubscribe(s *Subscription) (FutureMessage, message.OnMessage) {
	exOnUnSub := ds.ExpectMessageOnce(topics.Event, actions.Ack, actions.Unsubscribe, []byte(s.Name))

	go ds.Emit(message.New(topics.Event, actions.Unsubscribe, s.Name))

	return exOnUnSub, nil
}

// onSubscribe creates a expectMessage for subscription's ack and subscription's messages
// then emits a subscription message to the deepstream server
func (ds *DeepStream) onSubscribe(s *Subscription) (FutureMessage, FutureMessage) {
	slog := log.With(zap.String("name", s.Name))
	slog.Debug("DeepStream::onSubscribe")

	// on receive a message
	exOnMessage := ds.ExpectMessage(topics.Event, actions.Event, []byte(s.Name))

	// on subscribe ack
	exOnSub := ds.ExpectMessageOnce(topics.Event, actions.Ack, actions.Subscribe, []byte(s.Name))

	// send the subscription request
	slog.Debug("DeepStream::onSubscribe - emitting...")
	go ds.Emit(message.New(topics.Event, actions.Subscribe, s.Name))

	return exOnSub, exOnMessage
}

func (ds *DeepStream) onEmit(name string, data ...[]byte) {
	ds.Emit(message.New(topics.Event, actions.Event, name, data))
}

type FutureMessage chan *message.Message
type ExpectedMessage struct {
	*message.Message
	ShouldClose bool
	Channel     FutureMessage
}

func (ds *DeepStream) expectMessage(shouldClose bool, topic topics.Topic, action actions.Action, data ...[]byte) FutureMessage {
	mm := &message.Message{}
	mm.Parse(message.New(topic, action, data))

	// create a expected message with a parsed message and a
	// channel to send events when the expected message is found
	exm := ExpectedMessage{
		ShouldClose: shouldClose,
		Message:     mm,
		Channel:     make(FutureMessage),
	}

	// insert expected message into list
	ds.emu.Lock()
	defer ds.emu.Unlock()

	ds.expectedMessages = append(ds.expectedMessages, exm)

	return exm.Channel
}

// ExpectMessage checks every message for the expected one
// when found sends a event through a channel with the message
func (ds *DeepStream) ExpectMessage(topic topics.Topic, action actions.Action, data ...[]byte) FutureMessage {
	return ds.expectMessage(false, topic, action, data...)
}

// ExpectMessageOnce checks every message for the expected one
// when found sends a event through a channel with the message
// after the first event is sent, the channel is closed automatically
func (ds *DeepStream) ExpectMessageOnce(topic topics.Topic, action actions.Action, data ...[]byte) FutureMessage {
	return ds.expectMessage(true, topic, action, data...)
}
