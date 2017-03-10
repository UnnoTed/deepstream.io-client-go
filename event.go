package deepstream

import (
	"errors"
	"sync"

	"go.uber.org/zap"
)

var (
	ErrorSubscriptionExists       = errors.New("Error: subscription already exists.")
	ErrorSubscriptionDoesntExists = errors.New("Error: subscription doesn't exists.")
)

type Subscription struct {
	Name    string
	Channel FutureMessage
}

type bus struct {
	subscriptions map[string]*Subscription

	// handlers so it can be controlled by the deepstream pkg
	onSubscribe   func(*Subscription) (FutureMessage, FutureMessage)
	onUnsubscribe func(*Subscription) (FutureMessage, FutureMessage)
	onEmit        func(string, []byte)

	m *sync.RWMutex
}

func newEvent() *bus {
	e := &bus{}
	e.m = &sync.RWMutex{}
	e.subscriptions = map[string]*Subscription{}

	return e
}

func (e *bus) Subscribe(name string) (FutureMessage, FutureMessage, error) {
	slog := log.With(zap.String("name", name))
	slog.Debug("Event::Subscribe")

	// returns error when subscription already exists
	if _, ok := e.subscriptions[name]; ok {
		return nil, nil, ErrorSubscriptionExists
	}

	e.m.Lock()
	e.subscriptions[name] = &Subscription{
		Name: name,
	}
	e.m.Unlock()

	slog.Debug("Event::Subscribe Running onSubscribe")
	exp, onm := e.onSubscribe(e.subscriptions[name])
	return exp, onm, nil
}

func (e *bus) Unsubscribe(name string, cb func([]byte)) (FutureMessage, error) {
	e.m.Lock()
	defer e.m.Unlock()

	// returns error when subscription already exists
	if _, ok := e.subscriptions[name]; ok {
		return nil, ErrorSubscriptionDoesntExists
	}

	if e.onSubscribe != nil {
		e.onUnsubscribe(e.subscriptions[name])
	}

	delete(e.subscriptions, name)

	return nil, nil
}

func (e *bus) Emit(name string, data []byte) {
	e.onEmit(name, data)
}

func (e *bus) Listen(name string) {

}

func (e *bus) Unlisten(name string) {

}
