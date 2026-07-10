package bus

type api struct {

}

func newApi() *api {
	return &api {}
}

func (a *api) close() {}

func (a *api) register(n node) {}

func (a *api) unregister(n node) {}