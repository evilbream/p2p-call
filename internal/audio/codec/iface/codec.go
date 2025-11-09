package iface

type Encoder interface {
	Encode(pcm []int16) ([]byte, error)
}

type Decoder interface {
	Decode(encoded []byte) ([]int16, error)
}
