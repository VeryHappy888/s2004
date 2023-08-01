package utils

func PutInt24(i int) []byte {
	bArr := make([]byte, 3)
	bArr[2] = byte(i)
	bArr[1] = (byte)(i >> 8)
	bArr[0] = (byte)(i >> 16)
	return bArr
}
