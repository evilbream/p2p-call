package codec

type Encoder interface {
	EncodeFloat32(samples []float32) ([][]byte, error)
	EncodeFloat32Bytes(samples []byte) ([][]byte, error)
}

type Decoder interface {
	DecodeChunk(chunk []byte) ([]byte, error)
	DecodeChunks(chunks [][]byte) ([]byte, error)
}
