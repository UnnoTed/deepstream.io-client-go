package deepstream

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	cm    *sync.RWMutex
}

// Connect to a deepstream server
func (ds *DeepStream) Connect(url string) (FutureMessage, error) {
	if ds.Connection != nil {
		log.Info("You're already connected.")
		return nil, nil
	}

	if ds.cm == nil {
		ds.cm = &sync.RWMutex{}
	}

	if ds.emu == nil {
		ds.emu = &sync.RWMutex{}
	}

	ds.cm.Lock()
	ds.Closed = true
	ds.cm.Unlock()

	ds.url = url

	// insert ws protocol when not found
	if !strings.HasPrefix(url, "ws://") {
		url = "ws://" + url
	}

	// insert deepstream in the end when not found
	if !strings.HasSuffix(url, "/deepstream") {
		url += "/deepstream"
	}

	// sets the state to awaiting connection
	ds.state = consts.ConnectionStateAwaitingConnection

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

		ds.cm.Lock()
		ds.Closed = false
		ds.cm.Unlock()
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
	ds.cm.Lock()
	defer ds.cm.Unlock()

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
	ds.state = consts.ConnectionStateOpen
	ds.Emit(message.New(topics.Event, actions.Event, events.ConnectionStateChanged, string(state)))
}

// State returns the current connection's state
func (ds *DeepStream) State() consts.ConnectionState {
	if ds.Connection == nil {
		ds.state = consts.ConnectionStateAwaitingConnection
	}

	return ds.state
}

// Authenticate sends a auth request to the deepstream server
func (ds *DeepStream) Authenticate(data map[string]interface{}) (FutureMessage, error) {
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
	ds.cm.Lock()
	defer ds.cm.Unlock()

	log.Info("Closing connection...")
	if err := ds.Connection.Close(); err != nil {
		return err
	}

	ds.Closed = true
	log.Info("Connection closed.")

	return nil
}

func (ds *DeepStream) reader() {
	first := true

	for {
		if ds.Connection == nil {
			break
		}

		log.Debug("Waiting for next message...")
		var err error
		_, ds.LastMessage, err = ds.Connection.ReadMessage()
		log.Info("MESSAGE FROM SERVER", zap.String("msg", message.String(ds.LastMessage)))

		ds.cm.RLock()
		if ds.Closed && !first {
			ds.cm.RUnlock()
			return
		}

		ds.cm.RUnlock()
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
		m := &message.Message{}
		if err := m.Parse(ds.LastMessage); err != nil {
			log.Error(err.Error())
		}

		// log.Debug("Saving to last message...")

		log.Debug("Checking for expected message",
			zap.String("msg", fmt.Sprintf("%s", m.String())))

		var toRemove []int
		ds.emu.RLock()
		for i, ex := range ds.expectedMessages {
			if bytes.Equal(ex.Message.Topic, m.Topic) && bytes.Equal(ex.Message.Action, m.Action) {
				log.Debug("Expected message has equal topic and action")

				if len(ex.Message.Data) > 0 && len(ex.Message.Data) == len(m.Data) {
					log.Debug("Checking equal data")

					for i := range ex.Message.Data {
						if !bytes.Equal(ex.Message.Data[i].Bytes, m.Data[i].Bytes) {
							continue
						}
					}
				}

				log.Debug("Found expected message", zap.String("data", ex.Message.String()))
				ds.emu.RUnlock()
				ex.Channel <- m
				ds.emu.RLock()

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
		// remove unexpected messages
		for _, i := range toRemove {
			if len(ds.expectedMessages) == 1 {
				ds.expectedMessages = ds.expectedMessages[:i]

			} else {
				ds.expectedMessages = append(ds.expectedMessages[:i], ds.expectedMessages[i+1:]...)
			}
		}
		ds.emu.Unlock()
	}
}

func (ds *DeepStream) onUnsubscribe(s *Subscription) (FutureMessage, message.OnMessage) {
	ds.Emit(message.New(topics.Event, actions.Unsubscribe, s.Name))
	return nil, nil
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

func (ds *DeepStream) onEmit(name string, data []byte) {
	ds.Emit(message.New(topics.Event, actions.Event, name, string(data)))
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

	exm := ExpectedMessage{
		ShouldClose: shouldClose,
		Message:     mm,
		Channel:     make(FutureMessage),
	}

	ds.emu.Lock()
	ds.expectedMessages = append(ds.expectedMessages, exm)
	ds.emu.Unlock()

	return exm.Channel
}

func (ds *DeepStream) ExpectMessage(topic topics.Topic, action actions.Action, data ...[]byte) FutureMessage {
	return ds.expectMessage(false, topic, action, data...)
}

func (ds *DeepStream) ExpectMessageOnce(topic topics.Topic, action actions.Action, data ...[]byte) FutureMessage {
	return ds.expectMessage(true, topic, action, data...)
}
