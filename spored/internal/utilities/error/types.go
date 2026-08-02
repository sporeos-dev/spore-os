package error

type Code string
const (
	ConnectionFailure Code = "ConnectionFailure"
	Generic Code = "Generic"
	HandleInUse Code = "HandleInUse"
	HandshakeDenial Code = "HandshakeDenial"
	InitializationFailure Code = "InitializationFailure"
	Malformed Code = "Malformed"
	Missing Code = "Missing"
	NotApplicable Code = "NotApplicable"
	NotImplemented Code = "NotImplemented"
	NotPermitted Code = "NotPermitted"
	ReservedLanguage Code = "ReservedLanguage"
	Timeout Code = "Timeout"
	UserDenial Code = "UserDenial"
)

type Module string
const (
	Bus Module = "Bus"
	Hub Module = "Hub"
	Hyphae Module = "Hyphae"
	Manifest Module = "Manifest"
	Message Module = "Message"
	Node Module = "Node"
	Permission Module = "Permission"
	Policy Module = "Policy"
	Registry Module = "Registry"
	Spore Module = "Spore"
)