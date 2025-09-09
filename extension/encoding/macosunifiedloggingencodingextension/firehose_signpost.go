package macosunifiedloggingencodingextension

type FirehoseSignpost struct{}

func ParseFirehoseSignpost(data []byte, flags uint16) (FirehoseSignpost, []byte) {
	return FirehoseSignpost{}, data
}
