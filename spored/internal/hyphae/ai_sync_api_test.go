// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package hyphae

import (
	"fmt"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	sync_event "spored/internal/event/sync"
	"testing"
)

type syncAPITestHandler struct {
	*base.Base
	handle func(sync_event.ISynchronizedRequestContext)
}

func (h *syncAPITestHandler) OnSynchronizedRequest(ctx sync_event.ISynchronizedRequestContext) {
	h.handle(ctx)
}

func TestSynchronizedHyphaeCalls(t *testing.T) {
	const path = "/home/a folder/it's \"node\"\\binary\nnext"
	const pid = 1234

	calls := []struct {
		name      string
		command   string
		arg       string
		value     string
		response  string
		want      string
		hasOutput bool
		call      func() (string, bool)
	}{
		{"manifest", "manifest.read", "path", path, `content="schema"`, "schema", true, func() (string, bool) { return ManifestRead(path) }},
		{"binary hash", "binary.hash", "pid", "1234", `hash="binary-hash"`, "binary-hash", true, func() (string, bool) { return BinaryHash(pid) }},
		{"file hash", "file.hash", "path", path, `hash="file-hash"`, "file-hash", true, func() (string, bool) { return FileHash(path) }},
		{"spawn", "hspawn", "binary", path, "", "", false, func() (string, bool) { return "", Hspawn(path) }},
		{"kill", "hkill", "pid", "1234", "", "", false, func() (string, bool) { return "", HKill(pid) }},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			scenarios := []struct {
				name     string
				response func(cparser.ParsedMessage) string
				wantOK   bool
			}{
				{"success", func(req cparser.ParsedMessage) string {
					return fmt.Sprintf("~%s:%s %s ok capture=dev.sporeos.HYPHAE", req.Handle, req.Command, call.response)
				}, true},
				{"error", func(req cparser.ParsedMessage) string {
					return fmt.Sprintf(`~%s:%s error code=Generic.Hyphae what="failed"`, req.Handle, req.Command)
				}, false},
				{"cancelled", func(req cparser.ParsedMessage) string {
					return fmt.Sprintf("~%s:%s ok cancelled", req.Handle, req.Command)
				}, false},
				{"malformed", func(req cparser.ParsedMessage) string {
					return fmt.Sprintf("~%s:%s", req.Handle, req.Command)
				}, false},
			}
			if call.hasOutput {
				scenarios = append(scenarios, struct {
					name     string
					response func(cparser.ParsedMessage) string
					wantOK   bool
				}{"missing output", func(req cparser.ParsedMessage) string {
					return fmt.Sprintf("~%s:%s ok", req.Handle, req.Command)
				}, false})
			}
			scenarios = append(scenarios, struct {
				name     string
				response func(cparser.ParsedMessage) string
				wantOK   bool
			}{"missing response", nil, false})

			for _, scenario := range scenarios {
				t.Run(scenario.name, func(t *testing.T) {
					event.Init()
					handler := &syncAPITestHandler{
						Base: base.New().SetName("hyphae-test"),
						handle: func(ctx sync_event.ISynchronizedRequestContext) {
							if ctx.Origin() != "dev.sporeos.SPORE" {
								t.Errorf("origin = %q", ctx.Origin())
							}
							req, ok := cparser.ParseAndValidate(ctx.Message())
							if !ok {
								t.Fatalf("invalid request: %q", ctx.Message())
							}
							if req.Type != cparser.TypeRequest || req.Command != "dev.sporeos.HYPHAE."+call.command || req.Handle == "" {
								t.Errorf("unexpected request: %+v", req)
							}
							if got := req.Args[call.arg]; got != call.value {
								t.Errorf("%s = %q, want %q", call.arg, got, call.value)
							}
							if scenario.response != nil {
								ctx.Respond(message_event.Response(scenario.response(req), "dev.sporeos.HYPHAE", req.Handle, ctx.Origin()))
							}
						},
					}
					event.Subscribe(handler)
					t.Cleanup(func() { event.Unsubscribe(handler) })

					got, ok := call.call()
					if ok != scenario.wantOK {
						t.Fatalf("success = %v, want %v", ok, scenario.wantOK)
					}
					if ok && got != call.want {
						t.Errorf("result = %q, want %q", got, call.want)
					}
				})
			}
		})
	}
}

func TestManifestReadReturnsMultilineContent(t *testing.T) {
	event.Init()
	handler := &syncAPITestHandler{
		Base: base.New().SetName("hyphae-test"),
		handle: func(ctx sync_event.ISynchronizedRequestContext) {
			req, ok := cparser.ParseAndValidate(ctx.Message())
			if !ok {
				t.Fatal("invalid request")
			}
			raw := fmt.Sprintf("~%s:%s ok content=<<\nschema: SPORE/v0d3\nname: \"node\"\n>>", req.Handle, req.Command)
			ctx.Respond(message_event.Response(raw, "dev.sporeos.HYPHAE", req.Handle, ctx.Origin()))
		},
	}
	event.Subscribe(handler)
	t.Cleanup(func() { event.Unsubscribe(handler) })

	content, ok := ManifestRead("/home/node.manifest.spore.yaml")
	if !ok || content != "schema: SPORE/v0d3\nname: \"node\"" {
		t.Errorf("ManifestRead = %q, %v", content, ok)
	}
}
