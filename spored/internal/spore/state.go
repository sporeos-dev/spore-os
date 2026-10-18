// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"os"
	"path/filepath"
	"spored/internal/iface"
	"spored/internal/message"
	"spored/internal/pal"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"strings"
)

func (s *Spore) statePersist(request iface.Message) *error.Error {

	cast := request.Cast()
	key, ok := request.Arg("key")
	if !ok {
		return error.MissingArg("key", error.Spore).WithMessage(request)
	}
	filename, persistError := persistFileName(pal.DirectoryData(), key, cast)
	if persistError != nil {
		return persistError.WithMessage(request)
	}

	body, ok := request.Arg("body")
	if !ok {
		return error.MissingArg("body", error.Spore).WithMessage(request)
	}

	go func() {
		if persistError := persistFile(filename, []byte(body)); persistError != nil {
			s.bus.Response(persistError.WithMessage(request))
			return
		}
		s.bus.Response(
			message.Spore(
				request,
			),
		)
	}()

	return nil
}

func (s *Spore) stateRetrieve(request iface.Message) *error.Error {

	cast := request.Cast()
	key, ok := request.Arg("key")
	if !ok {
		return error.MissingArg("key", error.Spore).WithMessage(request)
	}
	filename, persistError := persistFileName(pal.DirectoryData(), key, cast)
	if persistError != nil {
		return persistError.WithMessage(request)
	}

	go func() {
		body, retrieveError := os.ReadFile(filename)
		if retrieveError != nil {
			if os.IsNotExist(retrieveError) {
				s.bus.Response(message.Spore(
					request, 
					out.Pair("body", ""),
					out.Flag("empty")))
				return
			}
			s.bus.Response(error.New(
				error.Generic,
				error.Spore,
				"unable to retrieve persisted state",
				out.Pair("err", retrieveError.Error())).
				WithMessage(request))
			return
		}

		s.bus.Response(
			message.Spore(
				request, out.Pair("body", string(body))))
	}()

	return nil
}

func persistFileName(dataDirectory string, key string, cast string) (string, *error.Error) {
	if !validPersistIdentifier(cast) || !validPersistIdentifier(key) {
		return "", error.New(error.Malformed, error.Spore, "invalid persisted state identifier")
	}
	return filepath.Join(dataDirectory, cast+"."+key+".persist"), nil
}

func validPersistIdentifier(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.ContainsAny(value, `/\\`)
}

func persistFile(filename string, body []byte) *error.Error {
	directory := filepath.Dir(filename)
	if persistError := os.MkdirAll(directory, 0700); persistError != nil {
		return error.New(
			error.Generic,
			error.Spore,
			"unable to create persisted state directory",
			out.Pair("err", persistError.Error()))
	}
	if persistError := os.Chmod(directory, 0700); persistError != nil {
		return error.New(
			error.Generic,
			error.Spore,
			"unable to secure persisted state directory",
			out.Pair("err", persistError.Error()))
	}
	if persistError := os.WriteFile(filename, body, 0600); persistError != nil {
		return error.New(
			error.Generic,
			error.Spore,
			"unable to persist state",
			out.Pair("err", persistError.Error()))
	}
	if persistError := os.Chmod(filename, 0600); persistError != nil {
		return error.New(
			error.Generic,
			error.Spore,
			"unable to secure persisted state",
			out.Pair("err", persistError.Error()))
	}
	return nil
}
