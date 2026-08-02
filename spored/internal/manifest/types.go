package manifest

type Start string
const (
	Auto Start = "auto"
	Lazy Start = "lazy"
	Manual Start = "manual"
)

type Space string
const (
	Spore Space = "spore"
	Hyphae Space = "hyphae"
	Any Space = "any"
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
