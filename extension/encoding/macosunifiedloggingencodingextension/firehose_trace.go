package macosunifiedloggingencodingextension

type FirehoseTrace struct {
	MessageData FirehoseItemData
}

func ParseFirehoseTrace(data []byte) (FirehoseTrace, []byte) {
	return FirehoseTrace{}, data
}
