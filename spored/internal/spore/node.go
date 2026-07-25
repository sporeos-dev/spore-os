package spore

import (
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

func (s *Spore) nodeList(request message.Message) *error.Error {

	go func() {
		nodeids := append([]string{s.manifest.ID}, s.nodes.GetNodes()...)
		s.respond(request, out.Array("nodes", nodeids))		
	}()

	return nil
}

func (s *Spore) nodeHelp(request message.Message) *error.Error {
	
	node, err := request.Arg("node")
	if err != nil {
		return err
	}

	go func() {
		var m *manifest.Manifest
		if node == s.manifest.ID {
			m = s.manifest
		} else {
			m = s.nodes.GetManifest(node)
		}
		if m == nil {
			err := error.New(
				error.Missing,
				error.Spore,
				"node manifest not loaded",
				out.Pair("node", node))
			s.respondError(request, err)
			return
		}

		s.respond(
			request,
			out.Array(m.ID,
				[]string{
					m.Name,
					m.Description,
				}),
			out.Pair("manifest", m.Path),
			out.Pair("binary", m.GetBinaryPath()),
			out.Array("API", m.CommandIds()),
			out.Array("Topics", m.TopicIds()))
	}()

	return nil
}

func (s *Spore) nodeState(request message.Message) *error.Error {
	
	go func() {
		err := error.New(
			error.NotImplemented,
			error.Spore,
			"node state not yet implemented")
		s.respondError(request, err)
	}()

	return nil
}

func (s *Spore) nodeInstall(request message.Message) *error.Error {
	
	path, err := request.Arg("path")
	if err != nil {
		return err
	}

	go func() {
		err := s.nodes.Install(path)
		if err != nil {
			s.respondError(request, err)
		} else {
			s.respond(request)
		}
	}()

	return nil
}

func (s *Spore) nodeUninstall(request message.Message) *error.Error {
	
	node, err := request.Arg("node")
	if err != nil {
		return err
	}

	go func() {
		err := s.nodes.Uninstall(node)
		if err != nil {
			s.respondError(request, err)
		} else {
			s.respond(request)
		}
	}()

	return nil
}

func (s *Spore) nodeSpawn(request message.Message) *error.Error {

	node, err := request.Arg("node")
	if err != nil {
		return err
	}

	go func() {
		err := s.nodes.Spawn(node)
		if err != nil {
			s.respondError(request, err)
		} else {
			s.respond(request)
		}
	}()

	return nil
}

func (s *Spore) nodeKill(request message.Message) *error.Error {

	node, err := request.Arg("node")
	if err != nil {
		return err
	}

	go func() {
		err := s.nodes.Kill(node)
		if err != nil {
			s.respondError(request, err)
		} else {
			s.respond(request)
		}
	}()

	return nil
}

