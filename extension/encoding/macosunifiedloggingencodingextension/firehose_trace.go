package macosunifiedloggingencodingextension

type FirehoseTrace struct{}

func ParseFirehoseTrace(data []byte) (FirehoseTrace, []byte) {
	return FirehoseTrace{}, data
}
