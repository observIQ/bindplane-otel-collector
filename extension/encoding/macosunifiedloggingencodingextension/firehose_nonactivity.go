package macosunifiedloggingencodingextension

type FirehoseNonActivity struct {
	PrivateStringsSize   uint16
	PrivateStringsOffset uint16
}

func ParseFirehoseNonActivity(data []byte, flags uint16) (FirehoseNonActivity, []byte) {
	return FirehoseNonActivity{}, data
}
