package manifest

func (m *Manifest) CommandIds() []string {
	ids := make([]string, 0, len(m.Api))
	for _, el := range m.Api {
		ids = append(ids, el.Name)
	}
	return ids
} 

func (m *Manifest) TopicIds() []string {
	ids := make([]string, 0, len(m.Topics))
	for _, el := range m.Topics {
		ids = append(ids, el.Name)
	}
	return ids
}

func (m *Manifest) ErrorIds() []string {
	ids := make([]string, 0, len(m.Errors))
	for _, el := range m.Errors {
		ids = append(ids, el.Name)
	}
	return ids
}

