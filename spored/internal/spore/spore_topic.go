package spore

import (
	"encoding/json"
	"errors"
	"spored/internal/message"
)

func (s *Spore) topicList(msg *message.Spore) (string, error) {
	nodeid := msg.GetArgumentIf("node", "n/a")

	var topics []string
	if nodeid == "dev.sporeos.SPORE" {
		if s.manifest == nil {
			return "", errors.New("SPORE manifest not loaded")
		}
		for _, topic := range s.manifest.Topics {
			topics = append(topics, topic.Name)
		}
	} else {
		topics = s.broadcaster.ListTopics(nodeid)
		if nodeid == "n/a" && s.manifest != nil {
			for _, topic := range s.manifest.Topics {
				topics = append(topics, topic.Name)
			}
		}
	}

	serialized, err := json.Marshal(topics)
	if err != nil {
		return "", err
	}
	return s.returnCapture(msg, map[string]string{"topics": string(serialized)}, []string{}), nil
}

func (s *Spore) topicHelp(msg *message.Spore) (string, error) {
	return s.returnCapture(msg, map[string]string{}, []string{}), nil
}

func (s *Spore) topicSubscribe(msg *message.Spore) (string, error) {
	topic, err := msg.GetArgument("topic")
	if err != nil {
		return "", err
	}
	s.broadcaster.Subscribe(msg.Cast(), topic)
	return s.returnCapture(msg, map[string]string{}, []string{}), nil
}

func (s *Spore) topicUnsubscribe(msg *message.Spore) (string, error) {
	topic, err := msg.GetArgument("topic")
	if err != nil {
		return "", err
	}
	s.broadcaster.Unsubscribe(msg.Cast(), topic)
	return s.returnCapture(msg, map[string]string{}, []string{}), nil
}

