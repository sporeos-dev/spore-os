package app_event

type IAppEntryHandler interface {
	OnAppEntry()
}

type IAppExitHandler interface {
	OnAppExit()
}
