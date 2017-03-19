package message

import (
	"errors"
	"log"
	"regexp"

	"github.com/gobwas/glob"
)

var ErrorTypeNotSupported = errors.New("Error: type not supported, use glob.Glob or *regexp.Regexp")

type Matcher struct {
	Glob    glob.Glob
	Pattern *regexp.Regexp
}

func NewMatcher(pattern interface{}) (*Matcher, error) {
	m := &Matcher{}

	switch pattern.(type) {
	case glob.Glob:
		m.Glob = pattern.(glob.Glob)
	case *regexp.Regexp:
		m.Pattern = pattern.(*regexp.Regexp)
	default:
		return nil, ErrorTypeNotSupported
	}

	return m, nil
}

func (m *Matcher) Matches(msg *Message) bool {
	log.Println("MESSAGE MATCHES", msg.String())
	if m.Empty() {
		return false

	} else if m.IsRegex() {
		return m.Pattern.Match(msg.Raw)
	}

	return m.Glob.Match(msg.StringData())
}

func (m *Matcher) IsGlob() bool {
	return m.Glob != nil && m.Pattern == nil
}

func (m *Matcher) IsRegex() bool {
	return m.Pattern != nil && m.Glob == nil
}

func (m *Matcher) Empty() bool {
	return !m.IsRegex() && !m.IsGlob()
}
