package macosunifiedloggingencodingextension

import "encoding/binary"

type FirehoseLoss struct {
	// TODO: consider using time.Time instead of uint64 once timesync is re-examined
	StartTime uint64
	EndTime   uint64
	Count     uint64
}

func ParseFirehoseLoss(data []byte) (FirehoseLoss, []byte) {
	firehoseLoss := FirehoseLoss{}

	data, startTime, _ := Take(data, 8)
	data, endTime, _ := Take(data, 8)
	data, count, _ := Take(data, 8)

	firehoseLoss.StartTime = binary.LittleEndian.Uint64(startTime)
	firehoseLoss.EndTime = binary.LittleEndian.Uint64(endTime)
	firehoseLoss.Count = binary.LittleEndian.Uint64(count)

	return firehoseLoss, data
}
