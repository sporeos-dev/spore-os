package error

type Code string
const (
	// Internal / meta
	UnknownFailure  Code = "UnknownFailure"
	SporeFailure    Code = "SporeFailure"
	ProtocolFailure Code = "ProtocolFailure"
	ConnectionFailure Code = "ConnectionFailure"

	HandshakeDenial Code = "HandshakeDenial"
	InstallationFailure Code = "InstallationFailure"
	RegistryFailure Code = "RegistryFailure"

	// General
	Generic            Code = "Generic"
	Fatal              Code = "Fatal"
	Timeout            Code = "Timeout"
	Busy               Code = "Busy"
	ResourcesExhausted Code = "ResourcesExhausted"
	Deprecated         Code = "Deprecated"
	Runtime            Code = "Runtime"
	Logic              Code = "Logic"
	ReservedKeyword    Code = "ReservedKeyword"

	// Route
	RouteNotFound       Code = "RouteNotFound"
	RouteNotConnected   Code = "RouteNotConnected"
	RouteNotAvailable   Code = "RouteNotAvailable"
	RouteNotAllowed     Code = "RouteNotAllowed"
	RouteNotImplemented Code = "RouteNotImplemented"

	// Message
	MessageNotValid  Code = "MessageNotValid"
	MessageMalformed Code = "MessageMalformed"

	// Arguments
	ArgumentMissing      Code = "RequiredArgumentMissing"
	ArgumentInvalidType  Code = "ArgumentInvalidType"
	ArgumentConflict     Code = "ArgumentConflict"
	ArgumentOutOfRange   Code = "ArgumentOutOfRange"
	ArgumentUnrecognized Code = "ArgumentUnrecognized"
	ArgumentDuplicated   Code = "ArgumentDuplicated"

	// Flags
	FlagConflict     Code = "FlagConflict"
	FlagUnrecognized Code = "FlagUnrecognized"
	FlagDuplicated   Code = "FlagDuplicated"

	// Handles
	HandleMissing Code = "HandleMissing"
	HandleInUse   Code = "HandleInUse"
	HandleExpired Code = "HandleExpired"
)
