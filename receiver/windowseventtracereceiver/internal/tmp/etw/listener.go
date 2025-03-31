package etw

type Listener struct {
	events chan *Event
}

func NewListener() *Listener {
	return &Listener{
		events: make(chan *Event),
	}
}

func (l *Listener) Events() <-chan *Event {
	return l.events
}

func (l *Listener) Start() error {
	return nil
}
