package base

import "sync"

// inject base
//
// type Object struct {
//   *base.Base
// }
//
// func New(name string, id string)

type IBase interface {
	Name() string
	Id() string
	Key() string
	Set(name string, id string) *Base
	SetName(name string) *Base
	SetId(id string) *Base
}

type Base struct {
	sync.RWMutex
	name string
	id string
}

func New() *Base {
	return &Base{
		name: "n/a",
		id: "n/a",
	}
}

// ibase
func (b *Base) Name() string { 
	b.RLock()
	defer b.RUnlock()
	return b.name 
}

func (b *Base) Id() string { 
	b.RLock()
	defer b.RUnlock()
	return b.id 
}

func (b *Base) Key() string { 
	b.RLock()
	defer b.RUnlock()
	return b.name + "." + b.id 
}

func (b *Base) Set(name string, id string) *Base { 
	b.Lock()
	defer b.Unlock()
	b.name = name; b.id = id 
	return b
}

func (b *Base) SetName(name string) *Base{ 
	b.Lock()
	defer b.Unlock()
	b.name = name 
	return b
}

func (b *Base) SetId(id string) *Base { 
	b.Lock()
	defer b.Unlock()
	b.id = id 
	return b
}
