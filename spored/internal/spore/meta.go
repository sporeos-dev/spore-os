// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"fmt"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"strconv"
	"strings"
)

func (s *Spore) help(request iface.Message) *error.Error {

	body := request.ArgIf("body", "")

	go func() {

		outs := s.getHelp(body, len(body))
		outs = append(outs, out.Pair("body", body))
		
		if len(outs) > 0 {
			s.bus.Response(
				message.Spore(
					request,
					outs...,
				),
			)
		
		} else {
			s.bus.Response(
				error.New(
					error.NoMatches,
					error.Spore,
					"no help matches found",
					out.Pair("body", body)).
					WithMessage(request))
		}

	}()

	return nil
}

func (s *Spore) state(request iface.Message) *error.Error {

	body := request.ArgIf("body", "")

	go func() {

		outs := s.getState(body, len(body))
		outs = append(outs, out.Pair("body", body))
		
		if len(outs) > 0 {
			s.bus.Response(
				message.Spore(
					request,
					outs...,
				),
			)
		
		} else {
			s.bus.Response(
				error.New(
					error.NoMatches,
					error.Spore,
					"no state found",
					out.Pair("body", body)).
					WithMessage(request))
		}

	}()

	return nil
}

func (s *Spore) list(request iface.Message) *error.Error {

	body := request.ArgIf("body", "")

	go func() {

		outs := s.getList(body, len(body))
		outs = append(outs, out.Pair("body", body))
		
		if len(outs) > 0 {
			s.bus.Response(
				message.Spore(
					request,
					outs...,
				),
			)
		
		} else {
			s.bus.Response(
				error.New(
					error.NoMatches,
					error.Spore,
					"no list found",
					out.Pair("body", body)).
					WithMessage(request))
		}

	}()

	return nil
}

func (s *Spore) complete(request iface.Message) *error.Error {

	body := request.ArgIf("body", "")

	go func() {

		outs := s.getComplete(body, len(body))
		
		if len(outs) > 0 {
			outs = append(outs, out.Pair("body", body))
			s.bus.Response(
				message.Spore(
					request,
					outs...,
				),
			)
		
		} else {
			s.bus.Response(
				error.New(
					error.NoMatches,
					error.Spore,
					"no completions found",
					out.Pair("body", body)).
					WithMessage(request))
		}

	}()

	return nil
}

func (s *Spore) hint(request iface.Message) *error.Error {

	body := request.ArgIf("body", "")
	cursorStr := request.ArgIf("cursor", fmt.Sprint(len(body)))
	cursor, _ := strconv.Atoi(cursorStr)

	go func() {

		outs := make([]out.IOut, 0)
		outHelps := s.getHelp(body, cursor)
		if len(outHelps) > 0 {
			outs = append(outs, outHelps...)
		}
		outStates := s.getState(body, cursor)
		if len(outStates) > 0 {
			outs = append(outs, outStates...)
		}
		outLists := s.getList(body, cursor)
		if len(outLists) > 0 {
			outs = append(outs, outLists...)
		}

		// if values exist, respond
		if len(outs) > 0 {
			outs = append(outs, out.Pair("body", body))
			s.bus.Response(
				message.Spore(
					request,
					outs...,
				),
			)

		// otherwise completions
		} else {
			outCompletes := s.getComplete(body, cursor)
			outCompletes = append(outCompletes, out.Pair("body", body))
			if len(outCompletes) > 0 {
				s.bus.Response(
					message.Spore(
						request,
						outCompletes...,
					),
				)

			} else {
				s.bus.Response(
					error.New(
						error.NoMatches,
						error.Spore,
						"no hint matches found",
						out.Pair("body", body)).
						WithMessage(request))
			}
		}
	}()

	return nil
}

func btos(b bool) string { if b { return "true" } else { return "false" } }

func (s *Spore) getHelp(body string, cursor int) []out.IOut {
	outs := make([]out.IOut, 0)
	parts, _ := parseBody(body, cursor)

	if len(parts) == 0 {
		return outs
	}
	
	m, errCode := s.matchNode(parts[0])
	switch errCode {
	case error.Collision:
		s.nodeCollisionError(parts[0])
		outs = append(outs, out.Pair("error", "node collision"))
	case error.OK:
		outs = append(outs, out.Pair("id", m.ID))
		outs = append(outs, out.Pair("name", m.Name))
		outs = append(outs, out.Pair("description", m.Description))
		outs = append(outs, out.Pair("trust", string(m.Trust)))
		outs = append(outs, out.Pair("schema", m.Schema))
		outs = append(outs, out.Pair("version", m.Version))
		outs = append(outs, out.Pair("app", m.GetBinaryPath()))
		outs = append(outs, out.Pair("launch", string(m.Launch)))
		outs = append(outs, out.Pair("namespace", string(m.Namespace)))
		outs = append(outs, out.Pair("witness", btos(m.Witness)))
		return outs
	default:
		// do nothing
	}

	m, c, errCode := s.matchCommand(parts[0])
	switch errCode {
		case error.Collision:
			s.commandCollisionError(parts[0])
			outs = append(outs, out.Pair("error", "command collision"))
		case error.OK:
			outs = append(outs, out.Pair("node", m.Name))
			outs = append(outs, out.Pair("command", c.Name))
			outs = append(outs, out.Pair("fully-qualified", m.ID + "." + c.Name))
			outs = append(outs, out.Pair("description", c.Description))
			outs = append(outs, out.Pair("risk", string(c.Risk)))
			outs = append(outs, out.Array("usage", c.Usage))
			if c.Inputs != nil && len(*c.Inputs) > 0 {
				inputs := make([]string, 0)
				for _, el := range *c.Inputs {
					inputs = append(inputs, el.Name)
				}
				outs = append(outs, out.Array("inputs", inputs))
			}
			if c.Outputs != nil && len(*c.Outputs) > 0 {
				outputs := make([]string, 0)
				for _, el := range *c.Outputs {
					outputs = append(outputs, el.Name)
				}
				outs = append(outs, out.Array("outputs", outputs))
			}
			if c.Notes != nil && len(*c.Notes) > 0 {
				outs = append(outs, out.Array("notes", *c.Notes))
			}
			return outs
		default:
			// do nothing
	}

	m, t, errCode := s.matchTopic(parts[0])
	switch errCode {
		case error.Collision:
			s.topicCollisionError(parts[0])
			outs = append(outs, out.Pair("error", "topic collision"))
		case error.OK:
			outs = append(outs, out.Pair("node", m.Name))
			outs = append(outs, out.Pair("topic", t.Name))
			outs = append(outs, out.Pair("fully-qualified", m.ID + "." + t.Name))
			outs = append(outs, out.Pair("description", t.Description))
			outs = append(outs, out.Pair("risk", string(t.Risk)))
			outs = append(outs, out.Array("usage", t.Usage))
			if t.Outputs != nil && len(*t.Outputs) > 0 {
				outputs := make([]string, 0)
				for _, el := range *t.Outputs {
					outputs = append(outputs, el.Name)
				}
				outs = append(outs, out.Array("outputs", outputs))
			}
			if t.Notes != nil && len(*t.Notes) > 0 {
				outs = append(outs, out.Array("notes", *t.Notes))
			}
			return outs
		default:
			// do nothing
	}

	return outs
}

func (s *Spore) getState(body string, cursor int) []out.IOut {
	outs := make([]out.IOut, 0)
	parts, _ := parseBody(body, cursor)

	if len(parts) == 0 {
		return outs
	}

	m, errCode := s.matchNode(parts[0])
	if errCode == error.OK {
		state, err := s.nodes.GetState(m.ID)
		connected := "false"
		if err != nil {
			connected = err.ArgIf("code", "unknowable")
		} else if state.Connected {
			connected = "true"
		}
		outs = append(outs, out.Pair("manifest", m.GetManifestPath()))
		outs = append(outs, out.Pair("binary", m.GetBinaryPath()))
		outs = append(outs, out.Pair("connected", connected))
		if state != nil && state.Pid > -1 {
			outs = append(outs, out.Pair("pid", strconv.Itoa(state.Pid)))
		}
		return outs
	}

	return outs
}

func (s *Spore) getList(body string, cursor int) []out.IOut {
	outs := make([]out.IOut, 0)
	parts, _ := parseBody(body, cursor)

	if len(parts) == 0 {
		return outs
	}

	m, errCode := s.matchNode(parts[0])
	if errCode == error.OK {
		// api
		api := make([]string, 0)
		if len(m.Api) > 0 {
			for _, el := range m.Api {
				api = append(api, el.Name)
			}
			outs = append(outs, out.Array("API", api))
		}
		
		// topics
		arr := make([]string, 0)
		if len(m.Topics) > 0 {
			for _, el := range m.Topics {
				arr = append(arr, el.Name)
			}
			outs = append(outs, out.Array("TOPICS", arr))
		}

		// errors
		errs := make([]string, 0)
		if len(m.Errors) > 0 {
			for _, el := range m.Errors {
				errs = append(errs, el.Name)
			}
			outs = append(outs, out.Array("ERRORS", errs))
		}

		// permissions
		perms := make([]string, 0)
		if len(m.Permissions) > 0 {
			for _, el := range m.Permissions {
				perms = append(perms, el.Name)
			}
			outs = append(outs, out.Array("PERMISSIONS", perms))
		}

		if len(outs) > 0 {
			return outs
		}
	}

	return outs
}

func (s *Spore) getComplete(body string, cursor int) []out.IOut {
	outs := make([]out.IOut, 0)
	parts, _ := parseBody(body, cursor)

	if len(parts) == 0 {
		nodes := s.nodes.GetNodes()
		nodes = append(nodes, "dev.sporeos.SPORE")
		outs = append(outs, out.Array("NODES", nodes))
		return outs
	}

	nodes := s.nodes.GetNodes()
	nodes = append(nodes, "dev.sporeos.SPORE")
	for _, node := range nodes {
		var m *manifest.Manifest
		if node == "dev.sporeos.SPORE" {
			m = s.manifest
		} else {
			m = s.nodes.GetManifest(node)
		}

		if m == nil {
			if strings.Contains(node, parts[0]) {
				outs = append(outs, out.Pair(node, "unknown node"))
			}
			continue
		}

		if strings.Contains(node, parts[0]) {
			outs = append(outs, out.Pair(node, m.Name))
		}

		for _, command := range m.Api {
			fqName := m.ID + "." + command.Name
			if strings.Contains(fqName, parts[0]) {
				outs = append(outs, out.Pair(command.Name, m.ID))
			}
		}

		for _, topic := range m.Topics {
			fqName := m.ID + "." + topic.Name
			if strings.Contains(fqName, parts[0]) {
				outs = append(outs, out.Pair(topic.Name, m.ID))
			}
		}
	}

	return outs
}

func (s *Spore) matchNode(part string) (*manifest.Manifest, error.Code) {
	if part == "SPORE" || part == "sporeos.SPORE" || part == "dev.sporeos.SPORE" {
		return s.manifest, error.OK
	}

	nodes := s.nodes.GetNodes()
	found := make([]string, 0)
	for _, node := range nodes {
		if node == part || strings.HasSuffix(node, "." + part) {
			found = append(found, node)
		}
	}

	switch len(found) {
	case 1:
		return s.nodes.GetManifest(found[0]), error.OK
	case 0:
		return nil, error.NoMatches
	default:
		return nil, error.Collision
	}
}

func (s *Spore) matchCommand(part string) (*manifest.Manifest, *manifest.Command, error.Code) {
	
	nodes := s.nodes.GetNodes()
	nodes = append(nodes, "dev.sporeos.SPORE")
	var fm *manifest.Manifest
	found := make([]manifest.Command, 0)
	for _, node := range nodes {
		var m *manifest.Manifest
		if node == "dev.sporeos.SPORE" {
			m = s.manifest
		} else {
			m = s.nodes.GetManifest(node)
		}
		for _, command := range m.Api {
			fqName := m.ID + "." + command.Name
			if fqName == part || strings.HasSuffix(fqName, "." + part) {
				fm = m
				found = append(found, command)
			}
		}
	}

	println("Found commands:", len(found))
	switch len(found) {
	case 1:
		return fm, &found[0], error.OK
	case 0:
		return nil, nil, error.NoMatches
	default:
		return nil, nil, error.Collision
	}
}

func (s *Spore) matchTopic(part string) (*manifest.Manifest, *manifest.Topic, error.Code) {
	
	nodes := s.nodes.GetNodes()
	nodes = append(nodes, "dev.sporeos.SPORE")
	found := make([]manifest.Topic, 0)
	var fm *manifest.Manifest
	for _, node := range nodes {
		var m *manifest.Manifest
		if node == "dev.sporeos.SPORE" {
			m = s.manifest
		} else {
			m = s.nodes.GetManifest(node)
		}
		for _, topic := range m.Topics {
			fqName := m.ID + "." + topic.Name
			if fqName == part || strings.HasSuffix(fqName, "." + part) {
				fm = m
				found = append(found, topic)
			}
		}
	}

	switch len(found) {
	case 1:
		return fm, &found[0], error.OK
	case 0:
		return nil, nil, error.NoMatches
	default:
		return nil, nil, error.Collision
	}
}

// parseBody splits body on spaces, keeping content inside '...', "...", [...], {...} and <<...>> intact,
// and also returns the token that the cursor (a caret index into body) falls within.
func parseBody(body string, cursor int) ([]string, string) {

	var parts []string
	var current strings.Builder
	var stack []string
	var quote byte

	tokStart := 0
	part := ""
	found := false

	flush := func(end int) {
		if current.Len() == 0 {
			return
		}
		s := current.String()
		parts = append(parts, s)
		if !found && cursor >= tokStart && cursor <= end+1 {
			part = s
			found = true
		}
		current.Reset()
	}

	n := len(body)
	i := 0
	for i < n {
		c := body[i]

		// inside a quote: only the matching quote closes it
		if quote != 0 {
			current.WriteByte(c)
			if c == quote {
				quote = 0
			}
			i++
			continue
		}

		// inside a bracket: its closer takes priority over opening a nested one
		if len(stack) > 0 {
			closer := stack[len(stack)-1]
			if strings.HasPrefix(body[i:], closer) {
				current.WriteString(closer)
				stack = stack[:len(stack)-1]
				i += len(closer)
				continue
			}
		}

		if current.Len() == 0 && c != ' ' {
			tokStart = i
		}

		switch {
		case len(stack) == 0 && c == ' ':
			flush(i - 1)
			i++
			tokStart = i

		case c == '\'' || c == '"':
			quote = c
			current.WriteByte(c)
			i++

		case c == '[':
			stack = append(stack, "]")
			current.WriteByte(c)
			i++

		case c == '{':
			stack = append(stack, "}")
			current.WriteByte(c)
			i++

		case c == '<' && i+1 < n && body[i+1] == '<':
			stack = append(stack, ">>")
			current.WriteString("<<")
			i += 2

		default:
			current.WriteByte(c)
			i++
		}
	}
	flush(n - 1)

	if !found && len(parts) > 0 {
		part = parts[len(parts)-1]
	}

	return parts, part
}

func (s *Spore) nodeCollisionError(part string) {
	nodes := s.nodes.GetNodes()
	found := make([]string, 0)
	for _, node := range nodes {
		if node == part || strings.HasSuffix(node, "." + part) {
			found = append(found, node)
		}
	}

	outs := make([]out.IOut, 0)
	for i, el := range found {
		outs = append(outs, out.Pair("node_" + strconv.Itoa(i + 1), el))
	}
	s.bus.Witness(message.Witness("Node collision", outs...))
}

func (s *Spore) commandCollisionError(part string) {
	nodes := s.nodes.GetNodes()
	nodes = append(nodes, "dev.sporeos.SPORE")
	f := make([]string, 0)
	for _, node := range nodes {
		var m *manifest.Manifest
		if node == "dev.sporeos.SPORE" {
			m = s.manifest
		} else {
			m = s.nodes.GetManifest(node)
		}
		for _, command := range m.Api {
			fqName := m.ID + "." + command.Name
			if command.Name == part || strings.HasSuffix(fqName, "." + part) {
				f = append(f, fqName)
			}
		}
	}

	outs := make([]out.IOut, 0)
	for i, el := range f {
		outs = append(outs, out.Pair("command_" + strconv.Itoa(i + 1), el))
	}
	s.bus.Witness(message.Witness("Command collision", outs...))
}

func (s *Spore) topicCollisionError(part string) {
	nodes := s.nodes.GetNodes()
	nodes = append(nodes, "dev.sporeos.SPORE")
	f := make([]string, 0)
	for _, node := range nodes {
		var m *manifest.Manifest
		if node == "dev.sporeos.SPORE" {
			m = s.manifest
		} else {
			m = s.nodes.GetManifest(node)
		}
		for _, topic := range m.Topics {
			fqName := m.ID + "." + topic.Name
			if topic.Name == part || strings.HasSuffix(fqName, "." + part) {
				f = append(f, fqName)
			}
		}
	}

	outs := make([]out.IOut, 0)
	for i, el := range f {
		outs = append(outs, out.Pair("topic_" + strconv.Itoa(i + 1), el))
	}
	s.bus.Witness(message.Witness("Topic collision", outs...))
}