package macosunifiedloggingencodingextension

type FirehoseLoss struct{}

func ParseFirehoseLoss(data []byte) (FirehoseLoss, []byte) {
	return FirehoseLoss{}, data
}
