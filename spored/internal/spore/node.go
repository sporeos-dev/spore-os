package spore

import (
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

func (s *Spore) nodeList(request icast) *error.Error {

	go func() {
		nodeids := s.nodes.GetNodes()
		s.respond(request, out.Array("nodes", nodeids))		
	}()

	return nil
}

func (s *Spore) nodeHelp(request icast) *error.Error {
	
	node, err := request.Arg("node")
	if err != nil {
		return err
	}

	go func() {
		manifest := s.nodes.GetManifest(node)
		if manifest == nil {
			err := error.New(
				error.Missing,
				error.Spore,
				"node manifest not loaded",
				out.Pair("node", node))
			s.respondError(request, err)
			return
		}
		
		response := map[string]any {
			"id": manifest.ID,
			"name": manifest.Name,
			"description": manifest.Description,
			"api": manifest.CommandIds(),
			"topics": manifest.TopicIds(),
		}

		s.respond(request, out.Object("nodehelp", response))
	}()

	return nil
}

func (s *Spore) nodeState(request icast) *error.Error {
	
	go func() {
		err := error.New(
			error.NotImplemented,
			error.Spore,
			"node state not yet implemented")
		s.respondError(request, err)
	}()

	return nil
}

func (s *Spore) nodeInstall(request icast) *error.Error {
	
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

func (s *Spore) nodeUninstall(request icast) *error.Error {
	
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

func (s *Spore) nodeSpawn(request icast) *error.Error {

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

func (s *Spore) nodeKill(request icast) *error.Error {

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

