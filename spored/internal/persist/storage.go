package persist

import (
	"fmt"
	"os"
	"path/filepath"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	"spored/internal/pal"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/witness"
	"strings"
)

var _ message_event.IRequestHandler = (*Storage)(nil)

type Storage struct {
	*base.Base
	dataDirectory string
}

func New() *Storage {
	return &Storage{
		Base: base.New().SetName("persistence"),
		dataDirectory: pal.DirectoryData(),
	}
}

func (s *Storage) Open() {
	event.Subscribe(s)
}

func (s *Storage) Close() {
	event.Unsubscribe(s)
}

func (s *Storage) OnRequest(ctx message_event.IRequestContext) {

	command := ctx.Command()
	message := ctx.Message()
	origin := ctx.Origin()
	handle := ctx.Handle()

	go func() {
		respondError := func(code error.Code, what string) {
			response := message_event.ResponseError(code, error.Persistence, what, handle, command, origin)
			event.Dispatch().Message.Notify(message_event.Raw(response.Message(), "dev.sporeos.SPORE"))
		}
		witnessError := func(code error.Code, what string, outs ...out.IOut) {
			witness.Error(code, error.Persistence, what, outs...)
		}

		pm, ok := cparser.ParseAndValidate(message)
		if !ok {
			// witness in ParseAndValidate
			respondError(error.Malformed, "bad message")
			return
		}		

		if command == "persist" || strings.HasSuffix(command, ".persist") {
			key, ok := pm.Args["key"]
			if !ok {
				witnessError(error.Missing, "missing arg", out.Pair("arg", key))
				respondError(error.Missing, "missing 'key'")
				return
			}
			body, ok := pm.Args["body"]
			if !ok {
				witnessError(error.Missing, "missing arg", out.Pair("arg", "body"))
				respondError(error.Missing, "missing 'body'")
				return
			}
			ok = s.statePersist(origin, key, body)
			if !ok {
				respondError(error.Generic, "persist failed")
				return
			}

			response := message_event.Raw(
				fmt.Sprintf("~%s:%s ok", handle, command), 
				"dev.sporeos.SPORE")
			event.Dispatch().Message.Notify(response)

		} else if command == "retrieve" || strings.HasSuffix(command, ".retrieve") {
			key, ok := pm.Args["key"]
			if !ok {
				witnessError(error.Missing, "missing arg", out.Pair("arg", "key"))
				respondError(error.Missing, "missing 'key'")
				return	
			}
			body, ok := s.stateRetrieve(origin, key)
			if !ok {
				response := message_event.Raw(
					fmt.Sprintf("~%s:%s ok empty", handle, command),
					"dev.sporeos.SPORE")
				event.Dispatch().Message.Notify(response)
				return
			}

			response := message_event.Raw(
				fmt.Sprintf("~%s:%s ok %s", handle, command, out.Pair("body", body)),
				"dev.sporeos.SPORE")
			event.Dispatch().Message.Notify(response)
		}
	}()
}

func (s *Storage) Api() []string {
	return []string{
		"dev.sporeos.SPORE.state.persist",
		"dev.sporeos.SPORE.state.retrieve",
	}
}

func (s *Storage) statePersist(node, key, body string) bool {
	filename, ok := persistFileName(s.dataDirectory, key, node)
	if !ok {
		return false
	}
	ok = persistFile(filename, []byte(body))
	if !ok {
		return false
	}
	return true
}

func (s *Storage) stateRetrieve(node, key string) (string, bool) {
	filename, ok := persistFileName(s.dataDirectory, key, node)
	if !ok {
		return "", false
	}
	body, err := os.ReadFile(filename)
	if err != nil {
		return "", false
	}
	return string(body), true
}

//
//
// private
// helpers
//

func persistFileName(dataDirectory string, key string, cast string) (string, bool) {
	if !validPersistIdentifier(cast) || !validPersistIdentifier(key) {
		witness.Error(
			error.Generic,
			error.Spore,
			"invalid identifier")
		return "", false
	}
	return filepath.Join(dataDirectory, cast+"."+key+".persist"), true
}

func validPersistIdentifier(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.ContainsAny(value, `/\\`)
}

func persistFile(filename string, body []byte) bool {
	directory := filepath.Dir(filename)
	if persistError := os.MkdirAll(directory, 0700); persistError != nil {
		witness.Error(
			error.Generic,
			error.Spore,
			"mkdir failed",
			out.Pair("err", persistError.Error()))
		return false
	}
	if persistError := os.Chmod(directory, 0700); persistError != nil {
		witness.Error(
			error.Generic,
			error.Spore,
			"chmod failed",
			out.Pair("err", persistError.Error()))
		return false
	}
	if persistError := os.WriteFile(filename, body, 0600); persistError != nil {
		witness.Error(
			error.Generic,
			error.Spore,
			"write failed",
			out.Pair("err", persistError.Error()))
		return false
	}
	if persistError := os.Chmod(filename, 0600); persistError != nil {
		witness.Error(
			error.Generic,
			error.Spore,
			"chmod failed",
			out.Pair("err", persistError.Error()))
		return false
	}
	return true
}





