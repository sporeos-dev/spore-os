package manifest

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
