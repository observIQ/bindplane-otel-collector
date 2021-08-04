package message

// Pipeline is used to transfer inbound and outbound messages.
type Pipeline struct {
	inbound  chan *Message
	outbound chan *Message
}

// NewPipeline returns a new pipeline with the supplied buffer size.
func NewPipeline(bufferSize int) *Pipeline {
	return &Pipeline{
		inbound:  make(chan *Message, bufferSize),
		outbound: make(chan *Message, bufferSize),
	}
}

// Inbound returns the inbound channel.
func (p *Pipeline) Inbound() chan *Message {
	return p.inbound
}

// Outbound returns the outbound channel.
func (p *Pipeline) Outbound() chan *Message {
	return p.outbound
}
