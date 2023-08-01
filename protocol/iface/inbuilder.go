package iface

type INodeOther interface {
	GetIqId() int32
}

// NodeBuilder
type NodeBuilder interface {
	IPromise
	// build node to ixxmp data
	Builder() ([]byte, error)
}
