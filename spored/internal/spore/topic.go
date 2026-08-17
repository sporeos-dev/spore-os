package spore

import (
	"spored/internal/iface"
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

func (s *Spore) topicList(request iface.Message) *error.Error {
	
	node := request.ArgIf("node", "all")

	go func() {
		
		topics := make([]string, 0)

		if node == "all" {

			topics = append(topics, s.manifest.TopicIds()...)
			for _, nodeid := range s.nodes.GetNodes() {
				manifest := s.nodes.GetManifest(nodeid)
				topics = append(topics, manifest.TopicIds()...)
			}

		} else if node == s.manifest.ID {

			topics = s.manifest.TopicIds()

		} else {

			manifest := s.nodes.GetManifest(node)
			if manifest == nil {
				s.bus.Response(
					error.New(
						error.Missing,
						error.Spore,
						"spore manifest not loaded").
						WithMessage(request))
				return
			}
			topics = manifest.TopicIds()

		}

		s.bus.Response(
			message.Spore(
				request,
				out.Array("topics", topics)))
	}()

	return nil
}

func (s *Spore) topicHelp(request iface.Message) *error.Error {

	topicid, ok := request.Arg("topic")
	if !ok {
		return error.MissingArg("topic", error.Spore).WithMessage(request)
	}

	go func() {

		for _, topic := range s.manifest.Topics {
			if topicid != topic.Name {
				continue
			}

			outputs := []string{}
			if topic.Outputs != nil {
				for _, output := range *topic.Outputs {
					outputs = append(outputs, output.Name+" ("+output.Type+"): "+output.Description)
				}
			}

			s.bus.Response(
				message.Spore(
					request,
					out.Array(topic.Name,
						[]string{
							topic.Description,
							s.manifest.ID,
						}),
					out.Array("Outputs", outputs)))
			return
		}

		for _, nodeid := range s.nodes.GetNodes() {
			
			manifest := s.nodes.GetManifest(nodeid)
			for _, topic := range manifest.Topics {
				if topicid != topic.Name {
					continue
				}
				
				outputs := []string{}
				if topic.Outputs != nil {
					for _, output := range *topic.Outputs {
						outputs = append(outputs, output.Name+" ("+output.Type+"): "+output.Description)
					}
				}

				s.bus.Response(
					message.Spore(
						request,
						out.Array(topic.Name,
							[]string{
								topic.Description,
								nodeid,
							}),
						out.Array("Outputs", outputs)))
				return
			}
		}

		s.bus.Response(
			error.New(
				error.Missing,
				error.Spore,
				"topic not found",
				out.Pair("topic", topicid)).
				WithMessage(request))
	}()

	return nil
}

func (s *Spore) topicState(request iface.Message) *error.Error {

	_, ok := request.Arg("topic")
	if !ok {
		return error.MissingArg("topic", error.Spore).WithMessage(request)
	}

	go func() {
		s.bus.Response(
			error.New(
				error.NotImplemented,
				error.Spore,
				"spore topic state not yet implemented").
				WithMessage(request))
	}()
	
	return nil
}

func (s *Spore) topicSubscribe(request iface.Message) *error.Error {

	topic, ok := request.Arg("topic")
	if !ok {
		return error.MissingArg("topic", error.Spore).WithMessage(request)
	}
	cast := request.Cast()

	go func() {
		// Check the topic exists in some installed node's manifest.
		found := false
		for _, nodeid := range s.nodes.GetNodes() {
			m := s.nodes.GetManifest(nodeid)
			if m == nil {
				continue
			}
			for _, t := range m.Topics {
				if t.Name == topic {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			err := error.New(
				error.Missing,
				error.Spore,
				"topic not found",
				out.Pair("topic", topic)).
				WithMessage(request)
			s.bus.Response(err)
			return
		}

		// Check the subscribing node has permission for this capability.
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
