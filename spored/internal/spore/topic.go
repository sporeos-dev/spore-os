// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"spored/internal/iface"
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

func (s *Spore) topicSubscribe(request iface.Message) *error.Error {

	topic, ok := request.Arg("topic")
	if !ok {
		return error.MissingArg("topic", error.Spore).WithMessage(request)
	}
	cast := request.Cast()

	go func() {
		// Existence and ambiguity of the (possibly abbreviated) topic name
		// are validated by bus.Subscribe itself.
		if !s.permissions.Can(cast, topic) {
			err := error.New(
				error.NotPermitted,
				error.Spore,
				"permission required to subscribe",
				out.Pair("node", cast),
				out.Pair("topic", topic)).
				WithMessage(request)
			s.bus.Response(err)
			return
		}

		err := s.bus.Subscribe(cast, topic)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
			return
		}

		s.bus.Response(message.Spore(request))
	}()

	return nil
}

func (s *Spore) topicUnsubscribe(request iface.Message) *error.Error {
	
	topic, ok := request.Arg("topic")
	if !ok {
		return error.MissingArg("topic", error.Spore).WithMessage(request)
	}
	cast := request.Cast()

	go func() {
		err := s.bus.Unsubscribe(cast, topic)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
			return
		}
		s.bus.Response(message.Spore(request))
	}()

	return nil
}
