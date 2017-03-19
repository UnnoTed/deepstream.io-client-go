package deepstream

import (
	"errors"
	"sync"

	"github.com/UnnoTed/deepstream.io-client-go/channels"
	"github.com/UnnoTed/deepstream.io-client-go/consts/actions"
	"github.com/UnnoTed/deepstream.io-client-go/consts/topics"
	"github.com/UnnoTed/deepstream.io-client-go/message"

	"go.uber.org/zap"
)

var (
	ErrorSubscriptionExists       = errors.New("Error: subscription already exists.")
	ErrorSubscriptionDoesntExists = errors.New("Error: subscription doesn't exists.")
	ErrorListenerDoesntExists     = errors.New("Error: subscription doesn't exists.")
)

type Subscription struct {
	Name         string
	Channel      channels.FutureMessage
	Pattern      *message.Matcher
	Expectations []int

	OnMessage *ExpectedMessage
	OnAck     *ExpectedMessage
}

type bus struct {
	subscriptions map[string]*Subscription
	listeners     map[string]*Subscription

	ds *DeepStream
	mu *sync.RWMutex
}

func newEvent() *bus {
	e := &bus{}
	e.mu = &sync.RWMutex{}
	e.subscriptions = map[string]*Subscription{}
	e.listeners = map[string]*Subscription{}

	return e
}

func (e *bus) GetSubscription(name string) (*Subscription, error) {
	sub, ok := e.subscriptions[name]
	if !ok {
		return nil, ErrorSubscriptionDoesntExists
	}

	return sub, nil
}

func (e *bus) GetListener(pattern string) (*Subscription, error) {
	lis, ok := e.listeners[pattern]
	if !ok {
		return nil, ErrorListenerDoesntExists
	}

	return lis, nil
}

// Subscribe creates a expectMessage for subscription's ack and subscription's messages
// then emits a subscription message to the deepstream server
func (e *bus) Subscribe(name string) (*ExpectedMessage, *ExpectedMessage, error) {
	slog := log.With(zap.String("name", name))
	slog.Debug("Event::Subscribe")

	// returns error when subscription already exists
	if _, ok := e.subscriptions[name]; ok {
		return e.subscriptions[name].OnAck, e.subscriptions[name].OnMessage, nil
		//return nil, nil, ErrorSubscriptionExists
	}

	e.mu.Lock()
	e.subscriptions[name] = &Subscription{
		Name: name,
	}
	e.mu.Unlock()

	slog.Debug("Event::Subscribe Running onEventSubscribe")
	s := e.subscriptions[name]

	slog.Debug("DeepStream::onSubscribe")

	// on receive a message
	exOnMessage, err := e.ds.ExpectMessage(topics.Event, actions.Event, []byte(s.Name))
	if err != nil {
		return nil, nil, err
	}

	// on subscribe ack
	exOnSub, err := e.ds.ExpectMessageOnce(topics.Event, actions.Ack, actions.Subscribe, []byte(s.Name))
	if err != nil {
		return nil, nil, err
	}

	// send the subscription request
	slog.Debug("DeepStream::onSubscribe - emitting...")
	go e.ds.Emit(message.New(topics.Event, actions.Subscribe, s.Name))

	s.OnMessage = exOnMessage
	s.OnAck = exOnSub

	return exOnSub, exOnMessage, nil
}

// Subscribe creates a ExpectMessageOnce for unsubscription's ack
// then emits a Unsubcribe message to the deepstream server
func (e *bus) Unsubscribe(name string) (*ExpectedMessage, error) {
	slog := log.With(zap.String("name", name))
	slog.Debug("Event::Unsubscribe")

	// returns error when subscription doesn't exists
	if _, ok := e.subscriptions[name]; !ok {
		return nil, ErrorSubscriptionDoesntExists
	}

	s := e.subscriptions[name]
	delete(e.subscriptions, name)

	slog.Debug("Event::Unsubscribe Running onEventUnsubscribe")
	slog.Debug("DeepStream::onUnsubscribe")

	exOnUnSub, err := e.ds.ExpectMessageOnce(topics.Event, actions.Ack, actions.Unsubscribe, []byte(s.Name))
	if err != nil {
		return nil, err
	}

	slog.Debug("DeepStream::onUnsubscribe - emitting...")
	go e.ds.Emit(message.New(topics.Event, actions.Unsubscribe, s.Name))

	return exOnUnSub, nil
}

func (e *bus) Emit(name string, data []byte) error {
	return e.ds.Emit(message.New(topics.Event, actions.Event, name, data))
}

func (e *bus) Listen(m *message.Matcher, pattern string) (*ExpectedMessage, *ExpectedMessage, error) {
	slog := log.With(zap.Any("matcher", m), zap.String("js pattern", pattern))
	slog.Debug("Event::Listen")

	// returns error when listener already exists
	if _, ok := e.listeners[pattern]; ok {
		e.listeners[pattern].OnAck.Channels = channels.New()

		slog.Debug("exists")
		go func(e *bus) {
			msg := &message.Message{}
			msg.Parse(message.New(topics.Event, actions.Event, pattern))
			e.listeners[pattern].OnAck.Channels.Send(msg)
		}(e)

		return e.listeners[pattern].OnAck, e.listeners[pattern].OnMessage, nil
		//return nil, nil, ErrorListenerExists
	}

	slog.Debug("creating")

	lis := &Subscription{
		Pattern: m,
		Name:    pattern,
	}

	e.mu.Lock()
	e.listeners[pattern] = lis
	e.mu.Unlock()

	slog.Debug("Event::Listen Running onEventListen")

	// on receive a message
	exOnMessage, err := e.ds.ExpectMessageWithMatcher(m, topics.Event, actions.Event)
	if err != nil {
		return nil, nil, err
	}

	// on listen ack
	exOnListen, err := e.ds.ExpectMessageOnce(topics.Event, actions.Ack, actions.Listen, []byte(lis.Name))
	if err != nil {
		return nil, nil, err
	}

	go e.ds.Emit(message.New(topics.Event, actions.Listen, []byte(lis.Name)))

	lis.OnMessage = exOnMessage
	lis.OnAck = exOnListen

	return exOnListen, exOnMessage, nil
}

func (e *bus) Unlisten(name string) (*ExpectedMessage, error) {
	slog := log.With(zap.String("name", name))
	slog.Debug("Event::Unlisten")

	// returns error when listener doesn't exists
	if _, ok := e.listeners[name]; !ok {
		return nil, ErrorListenerDoesntExists
	}

	lis := e.listeners[name]
	delete(e.listeners, name)

	slog.Debug("Event::Unlisten Running onEventUnlisten")

	// on unlisten ack
	exOnUnlisten, err := e.ds.ExpectMessageOnce(topics.Event, actions.Ack, actions.Unlisten, []byte(lis.Name))
	if err != nil {
		return nil, err
	}

	go e.ds.Emit(message.New(topics.Event, actions.Ack, actions.Unlisten, []byte(lis.Name)))

	return exOnUnlisten, err
}
