package convert

import "slices"

func IsFrameSizeValid(sampleRate, frameSize int) bool {
	switch sampleRate {
	case 48000:
		switch frameSize {
		case 120, 240, 480, 960, 1920, 2880:
			return true
		}
	case 16000:
		switch frameSize {
		case 40, 80, 160, 320, 640, 960:
			return true
		}
	case 8000:
		switch frameSize {
		case 20, 40, 80, 160, 320, 480:
			return true
		}
	default:
		// Generic check: frameSize corresponds to 2.5 /5 /10 /20 /40 /60 ms
		ms25 := sampleRate / 400
		valid := []int{ms25, ms25 * 2, ms25 * 4, ms25 * 8, ms25 * 16, ms25 * 24}
		if slices.Contains(valid, frameSize) {
			return true
		}
	}
	return false
}
