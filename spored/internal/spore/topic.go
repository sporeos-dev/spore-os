package spore

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

func (s *Spore) topicList(request message.Message) *error.Error {
	
	node := request.ArgIf("node", "all")

	go func() {
		
		topics := make([]string, 0)

		if node == "all" {

			for _, nodeid := range s.nodes.GetNodes() {
				manifest := s.nodes.GetManifest(nodeid)
				topics = append(topics, manifest.TopicIds()...)
			}

		} else {

			manifest := s.nodes.GetManifest(node)
			if manifest == nil {
				err := error.New(
					error.Missing,
					error.Spore,
					"spore manifest not loaded")
				s.respondError(request, err)
				return
			}
			topics = manifest.TopicIds()

		}

		s.respond(request, out.Array("topics", topics))
	}()

	return nil
}

func (s *Spore) topicHelp(request message.Message) *error.Error {

	topicid, err := request.Arg("topic")
	if err != nil {
		return err
	}

	go func() {

		for _, nodeid := range s.nodes.GetNodes() {
			
			manifest := s.nodes.GetManifest(nodeid)
			for _, topic := range manifest.Topics {
				if topicid != topic.Name {
					continue
				}
				
				response := map[string]any {
					"id": topic.Name,
					"name": topic.Name,
					"description": topic.Description,
					"node": nodeid,
					"outputs": topic.Outputs,
				}

				s.respond(request, out.Object("topichelp", response))
				return
			}
		}

		err := error.New(
			error.Missing,
			error.Spore,
			"topic not found",
			out.Pair("topic", topicid))
		s.respondError(request, err)
	}()

	return nil
}

func (s *Spore) topicState(request message.Message) *error.Error {

	_, err := request.Arg("topic")
	if err != nil {
		return err
	}

	go func() {
		err := error.New(
			error.NotImplemented,
			error.Spore,
			"spore topic state not yet implemented")
		s.respondError(request, err)
	}()
	return nil
}

func (s *Spore) topicSubscribe(request message.Message) *error.Error {

	topic, err := request.Arg("topic")
	if err != nil {
		return err
	}
	cast := request.Cast()
	
	go func() {
		err := s.bus.Subscribe(cast, topic)
		if err != nil {
			s.respondError(request, err)
			return
		}
		s.respond(request)
	}()

	return nil
}

func (s *Spore) topicUnsubscribe(request message.Message) *error.Error {
	
	topic, err := request.Arg("topic")
	if err != nil {
		return err
	}
	cast := request.Cast()

	go func() {
		err := s.bus.Unsubscribe(cast, topic)
		if err != nil {
			s.respondError(request, err)
			return
		}
		s.respond(request)
	}()

	return nil
}
