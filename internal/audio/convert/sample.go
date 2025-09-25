package convert

import (
	"encoding/binary"
	"math"
	"unsafe"
)

func Float32ToInt16(src []float32) []int16 {
	dst := make([]int16, len(src))
	for i, v := range src {
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		dst[i] = int16(v * 32767)
	}
	return dst
}

func Int16ToFloat32(src []int16) []float32 {
	dst := make([]float32, len(src))
	for i, v := range src {
		dst[i] = float32(v) / 32767.0
	}
	return dst
}

// FastBytesToFloat32 converts []byte to []float32 using unsafe pointer conversion.
// Note: This method assumes that the input byte slice length is a multiple of 4.
func FastBytesToFloat32(src []byte) []float32 {
	return (*[4096]float32)(unsafe.Pointer(&src[0]))[:len(src)/4]
}

// BytesToFloat32 safely converts []byte to []float32 using binary encoding
func BytesToFloat32(src []byte) []float32 {
	if len(src)%4 != 0 {
		src = src[:len(src)-(len(src)%4)]
	}

	dst := make([]float32, len(src)/4)
	for i := 0; i < len(dst); i++ {
		bits := binary.LittleEndian.Uint32(src[i*4 : i*4+4])
		dst[i] = *(*float32)(unsafe.Pointer(&bits))
	}
	return dst
}

// Int16ToBytes convert int16 sample to byte (Little Endian)
func Int16ToBytes(src []int16) []byte {
	dst := make([]byte, len(src)*2)
	for i, v := range src {
		binary.LittleEndian.PutUint16(dst[i*2:i*2+2], uint16(v))
	}
	return dst
}

func Float32ToBytes(data []float32) []byte {
	buf := make([]byte, len(data)*4)
	for i, f := range data {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}
