package permission

import (
	"fmt"
	"slices"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	manifest_event "spored/internal/event/manifest"
	message_event "spored/internal/event/message"
	sync_event "spored/internal/event/sync"
	"spored/internal/hyphae"
	"spored/internal/manifest"
	"spored/internal/spore"
	"spored/internal/utilities/error"
	"spored/internal/utilities/message"
	"spored/internal/utilities/out"
	"spored/internal/witness"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var _ message_event.IRequestHandler = (*Allowance)(nil)
var _ manifest_event.IManifestAnnouncementHandler = (*Allowance)(nil)
var _ manifest_event.IManifestWithdrawalHandler = (*Allowance)(nil)

//
//
// temporary
// grants
//

type temporaryGrant struct {
	node string
	capability string
	when time.Time
	token string
}

func makeGrant(node, capability string) temporaryGrant {
	return temporaryGrant{
		node: node,
		capability: capability,
		when: time.Now(),
		token: makeToken(),
	}
}

//
// 
// permission management
// "allowance"
//

type Allowance struct {
	*base.Base
	mu sync.RWMutex
	grants map[string][]string
	tokens []temporaryGrant

	protectedCapabilities []string
	adminCapabilities []string
	systemNodes []string
}

func New() *Allowance {
	return &Allowance{
		Base: base.New().SetName("allowance"),
		grants: make(map[string][]string),
		tokens: make([]temporaryGrant, 0),
	}
}

func (a *Allowance) Open() {
	event.Subscribe(a)
}

func (a *Allowance) Close() {
	event.Unsubscribe(a)
}

func (a *Allowance) Can(node, capability, msg string) bool {
	a.mu.Lock()

	if slices.Contains(a.protectedCapabilities, capability) {
		a.mu.Unlock()
		return false
	} else if slices.Contains(a.adminCapabilities, capability) {
		a.mu.Unlock()
		confirm, ok := hyphae.Confirm(node, capability)
		if !ok {
			witness.Error(error.NotPermitted, error.Permission, "confirmation attempt failed")
			return false
		}
		return confirm
	} else if slices.Contains(a.grants[node], capability) {
		a.mu.Unlock()
		return true
	} else if slices.Contains(a.systemNodes, node) {
		a.mu.Unlock()
		return true
	} 

	if !message.HasArg(msg, "permission.token") {
		if message.HasFlag(msg, "request.permission") {
			a.mu.Unlock()
			token, ok := spore.RequestPermission(node, capability)
			if !ok {
				return false
			} else {
				msg = message.RemoveFlag(msg, "request.permission")
				return a.Can(node, capability, msg + " permission.token=" + token)
			}
		} else {
			a.mu.Unlock()
			return false
		}
	}

	a.tokens = slices.DeleteFunc(a.tokens, func(t temporaryGrant) bool {
		return time.Since(t.when) >= time.Second * 30
	})

	for i, t := range a.tokens {
		if t.node == node && 
		   t.capability == capability && 
		   hasToken(msg, t.token) &&
		   time.Since(t.when) < time.Second * 30 {
			a.tokens = slices.Delete(a.tokens, i, i+1)
			a.mu.Unlock()
			return true
		}
	}

	a.mu.Unlock()
	return false
}

// irequesthandler
func (a *Allowance) OnRequest(ctx message_event.IRequestContext) {

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
			witness.Error(code, error.Mycelium, what, outs...)
		}

		pm, ok := cparser.ParseAndValidate(message)
		if !ok {
			// witnessed in ParseAndValidate
			respondError(error.Malformed, "bad message")
			return
		}

		if command == "request" || strings.HasSuffix(command, ".request") {

			capability, ok := pm.Args["capability"]
			if !ok {
				witnessError(error.Missing, "missing arg", out.Pair("arg", "capability"))
				respondError(error.Missing, "missing 'capability'")
				return
			}

			reasons, ok := pm.Args["reasons"]
			if !ok {
				reasons = "[ \"No reasons provided\", \"- as such, we suggest to decline\" ]"
			}

			// setup
			hcmd := "dev.sporeos.shell.confirm"
			nhandle := makeHandle()
			ques := fmt.Sprintf("Do you permit [%s] to [%s]?", origin, capability)
			raw := fmt.Sprintf("%s ~%s body=\"%s\" lines=%s", hcmd, nhandle, ques, reasons)

			// send & receive
			sreq := sync_event.Request(raw, "dev.sporeos.SPORE")
			event.Dispatch().SynchronizedRequest.Notify(sreq)
			srep := sreq.Response()
			if srep == nil {
				witness.Error(error.Missing, error.Permission, "missing confirmation response")
				respondError(error.Missing, "missing confirmation response")
				return
			}

			// parse & return
			pm, ok := cparser.ParseAndValidate(srep.Message())
			if !ok {
				// witnessed in ParseAndValidate
				respondError(error.Malformed, "bad response")
				return
			}

			if slices.Contains(pm.Flags, "error") {
				// witnessed in Incoming response
				respondError(error.Generic, "response failure")
				return
			
			} else if slices.Contains(pm.Flags, "cancelled") {
				// witnessed in Incoming response
				respondError(error.Generic, "cancelled")
				return
			
			} else if slices.Contains(pm.Flags, "no") {
				// witnessed in Incoming response
				response := message_event.Raw(
					fmt.Sprintf("~%s:%s ok no", handle, command),
					"dev.sporeos.SPORE")
				event.Dispatch().Message.Notify(response)
				return

			} else if slices.Contains(pm.Flags, "yes") {

				// witnessed in Incoming response
				tgrant := makeGrant(origin, capability)
				{
					a.mu.Lock()
					a.tokens = append(a.tokens, tgrant)
					a.mu.Unlock()
				}
				response := message_event.Raw(
					fmt.Sprintf("~%s:%s ok permission.token=%s", handle, command, tgrant.token),
					"dev.sporeos.SPORE")
				event.Dispatch().Message.Notify(response)
				return
			}

		} else if command == "grant" || strings.HasSuffix(command, ".grant") {

			node, ok := pm.Args["node"]
			if !ok {
				witnessError(error.Missing, "missing arg", out.Pair("arg", "node"))
				respondError(error.Missing, "missing 'node'")
				return
			}
			capability, ok := pm.Args["capability"]
			if !ok {
				witnessError(error.Missing, "missing arg", out.Pair("arg", "capability"))
				respondError(error.Missing, "missing 'capability'")
				return
			}

			{
				a.mu.Lock()
				if slices.Contains(a.grants[node], capability) {
					a.mu.Unlock()
					witnessError(error.Collision, "already granted")
					respondError(error.Collision, "already granted")
					return
				}
				a.grants[node] = append(a.grants[node], capability)
				a.mu.Unlock()
			}

			response := message_event.Raw(
				fmt.Sprintf("~%s:%s ok", handle, command),
				"dev.sporeos.SPORE")
			event.Dispatch().Message.Notify(response)

		} else if command == "revoke" || strings.HasSuffix(command, ".revoke") {

			node, ok := pm.Args["node"]
			if !ok {
				witnessError(error.Missing, "missing arg", out.Pair("arg", "node"))
				respondError(error.Missing, "missing 'node'")
				return
			}
			capability, ok := pm.Args["capability"]
			if !ok {
				witnessError(error.Missing, "missing arg", out.Pair("arg", "capability"))
				respondError(error.Missing, "missing 'capability'")
				return
			}

			{
				a.mu.Lock()
				if !slices.Contains(a.grants[node], capability) {
					a.mu.Unlock()
					witnessError(error.Missing, "not granted")
					respondError(error.Missing, "not granted")
					return
				}
				a.grants[node] = slices.Delete(a.grants[node], slices.Index(a.grants[node], capability), slices.Index(a.grants[node], capability)+1)
				a.mu.Unlock()
			}

			response := message_event.Raw(
				fmt.Sprintf("~%s:%s ok", handle, command),
				"dev.sporeos.SPORE")
			event.Dispatch().Message.Notify(response)
		} 
	}()
}

func (a *Allowance) Api() []string {
	return []string{
		"dev.sporeos.SPORE.permission.request",
		"dev.sporeos.SPORE.permission.grant",
		"dev.sporeos.SPORE.permission.revoke",
	}
}

// imanifestannouncement
// imanifestwithdrawal
func (a *Allowance) OnManifestAnnouncement(ctx manifest_event.IManifestAnnouncementContext) {

	man := ctx.Manifest()

	a.mu.Lock()
	defer a.mu.Unlock()

	switch man.Trust {
	case manifest.System:
		a.systemNodes = append(a.systemNodes, man.ID)
	}

	for _, el := range man.Api {
		name := man.ID + "." + el.Name
		switch el.Risk {
		case manifest.Protected:
			a.protectedCapabilities = append(a.protectedCapabilities, name)
		case manifest.Admin:
			a.adminCapabilities = append(a.adminCapabilities, name)
		}
	}

	for _, el := range man.Topics {
		name := man.ID + "." + el.Name
		switch el.Risk {
		case manifest.Protected:
			a.protectedCapabilities = append(a.protectedCapabilities, name)
		case manifest.Admin:
			a.adminCapabilities = append(a.adminCapabilities, name)
		}
	}
}

func (a *Allowance) OnManifestWithdrawal(ctx manifest_event.IManifestWithdrawalContext) {

	man := ctx.Manifest()

	a.mu.Lock()
	defer a.mu.Unlock()

	switch man.Trust {
	case manifest.System:
		for i, id := range a.systemNodes {
			if id == man.ID {
				a.systemNodes = append(a.systemNodes[:i], a.systemNodes[i+1:]...)
				break
			}
		}
	}

	for _, el := range man.Api {
		name := man.ID + "." + el.Name
		switch el.Risk {
		case manifest.Protected:
			a.protectedCapabilities = slices.Delete(a.protectedCapabilities, slices.Index(a.protectedCapabilities, name), slices.Index(a.protectedCapabilities, name)+1)
		case manifest.Admin:
			a.adminCapabilities = slices.Delete(a.adminCapabilities, slices.Index(a.adminCapabilities, name), slices.Index(a.adminCapabilities, name)+1)
		}
	}

	for _, el := range man.Topics {
		name := man.ID + "." + el.Name
		switch el.Risk {
		case manifest.Protected:
			a.protectedCapabilities = slices.Delete(a.protectedCapabilities, slices.Index(a.protectedCapabilities, name), slices.Index(a.protectedCapabilities, name)+1)
		case manifest.Admin:
			a.adminCapabilities = slices.Delete(a.adminCapabilities, slices.Index(a.adminCapabilities, name), slices.Index(a.adminCapabilities, name)+1)
		}
	}
}

//
//
// private
// helper
//

var index atomic.Int64
func makeHandle() string {
	return fmt.Sprintf("hyphae-%d", index.Add(1))
}
