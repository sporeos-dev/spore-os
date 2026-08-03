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
		s.bus.Response(message.Spore(request, out.Array("nodes", nodeids)))
	}()

	return nil
}

func (s *Spore) nodeHelp(request message.Message) *error.Error {
	
	node, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	go func() {
		var m *manifest.Manifest
		if node == s.manifest.ID {
			m = s.manifest
		} else {
			m = s.nodes.GetManifest(node)
		}
		if m == nil {
			s.bus.Response(
				error.New(
					error.Missing,
					error.Spore,
					"node manifest not loaded",
					out.Pair("node", node)).
					WithMessage(request))
			return
		}

		s.bus.Response(
			message.Spore(
				request,
				out.Array(m.ID,
					[]string{
						m.Name,
						m.Description,
					}),
				out.Pair("manifest", m.Path),
				out.Pair("binary", m.GetBinaryPath()),
				out.Array("API", m.CommandIds()),
				out.Array("Topics", m.TopicIds())))
	}()

	return nil
}

func (s *Spore) nodeState(request message.Message) *error.Error {
	
	node, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	go func() {
		state, err := s.nodes.GetState(node)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
			return
		}

		s.bus.Response(message.Spore(request, state...))
	}()

	return nil
}

func (s *Spore) nodeInstall(request message.Message) *error.Error {
	
	path, ok := request.Arg("path")
	if !ok {
		return error.MissingArg("path", error.Spore).WithMessage(request)
	}

	go func() {
		err := s.nodes.Install(path)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
		} else {
			s.bus.Response(message.Spore(request))
		}
	}()

	return nil
}

func (s *Spore) nodeUninstall(request message.Message) *error.Error {
	
	node, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	go func() {
		err := s.nodes.Uninstall(node)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
		} else {
			s.bus.Response(message.Spore(request))
		}
	}()

	return nil
}

func (s *Spore) nodeSpawn(request message.Message) *error.Error {

	node, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	go func() {
		err := s.nodes.Spawn(node)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
		} else {
			s.bus.Response(message.Spore(request))
		}
	}()

	return nil
}

func (s *Spore) nodeKill(request message.Message) *error.Error {

	node, ok := request.Arg("node")
	if !ok {
		return error.MissingArg("node", error.Spore).WithMessage(request)
	}

	go func() {
		err := s.nodes.Kill(node)
		if err != nil {
			s.bus.Response(err.WithMessage(request))
		} else {
			s.bus.Response(message.Spore(request))
		}
	}()

	return nil
}
