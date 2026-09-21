// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

type Launch string
const (
	Auto Launch = "auto"
	Lazy Launch = "lazy"
	Manual Launch = "manual"
)

type Namespace string
const (
	Spore Namespace = "spore"
	Hyphae Namespace = "hyphae"
	Any Namespace = "any"
)

type Trust string
const (
	System Trust = "system"
	Developer Trust = "developer"
	Trusted Trust = "trusted"
	StandardTrust Trust = "standard"
	Untrusted Trust = "untrusted"
)

type Risk string
const (
	Benign Risk = "benign"
	StandardRisk Risk = "standard"
	Personal Risk = "personal"
	Secret Risk = "secret"
	Protected Risk = "protected"
)
