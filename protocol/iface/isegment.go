package iface

// SegmentInputProcessor 处理tcp 输入流
type SegmentInputProcessor interface {
	ReadInputSegmentData() ([]byte, error)
}

// SegmentOutputProcessor 处理tcp 输出流
type SegmentOutputProcessor interface {
	WriteSegmentOutputData([]byte) error
}

// SegmentProcessor
type SegmentProcessor interface {
	SegmentInputProcessor
	SegmentOutputProcessor
}
