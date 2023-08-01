package newxxmp

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"ws-go/protocol/iface/ixxmp"
)

type Nodes []*Node

func (n *Nodes) AddNode(node *Node) {
	if len(*n) == 0 {
		*n = make([]*Node, 0)
	}
	*n = append(*n, node)
}

// EmptyNode is EmptyNode
func EmptyNode(tag string, objs ...interface{}) *Node {
	var attrs Attributes
	var children Nodes
	var data []byte
	// init
	attrs = make([]Attribute, 0)
	children = make([]*Node, 0)
	data = make([]byte, 0)
	// if objs is empty init attrs and children
	if len(objs) != 0 {
		for _, obj := range objs {
			switch obj.(type) {
			case Attributes:
				attrs = obj.(Attributes)
			case *Attributes:
				attrs = *obj.(*Attributes)
			case Nodes:
				children = obj.(Nodes)
			case *Nodes:
				children = *obj.(*Nodes)
			case *Node:
				children.AddNode(obj.(*Node))
			case Attribute:
				attribute := obj.(Attribute)
				attrs.AddAttr(attribute.Key(), attribute.Value())
			case []byte:
				data = obj.([]byte)
			}
		}
	}
	return &Node{Tag: tag, Attributes: attrs, Children: children, Data: data}
}

// Node
type Node struct {
	TagToken   ixxmp.IToken
	Tag        string
	Attributes Attributes
	Children   Nodes
	Data       []byte
}

func (n *Node) CheckNil() bool {
	return n.Attributes == nil
}
func (n *Node) EqualTag(tag string) bool {
	return strings.Compare(n.Tag, tag) == 0
}
func (n *Node) GetTag() string {
	return n.Tag
}
func (n *Node) GetChildren() []*Node {
	return n.Children
}
func (n *Node) GetChildrenIndex(i int) *Node {
	if n.Children == nil || len(n.Children) < i {
		return nil
	}
	return n.Children[i]
}
func (n *Node) GetChildrenByTag(tag string) *Node {
	for _, child := range n.Children {
		if strings.Compare(child.Tag, tag) == 0 {
			return child
		}
	}
	return nil
}
func (n *Node) GetAttributeByValue(s string) string {
	for _, attribute := range n.Attributes {
		if strings.Compare(attribute.Key(), s) == 0 {
			return attribute.Value()
		}
	}
	return ""
}
func (n *Node) GetAttribute(s string) *Attribute {
	for _, attribute := range n.Attributes {
		if strings.Compare(attribute.Key(), s) == 0 {
			return &attribute
		}
	}
	return nil
}
func (n *Node) SetData(d []byte) {
	n.Data = d
}
func (n *Node) GetData() []byte {
	// 需要进行截断
	if n.Data[0] == 0xfc {
		return n.Data[2:]
	}
	if n.Data[0] == 0xfd {
		return n.Data[4:]
	}
	return n.Data
}
func (n *Node) From(list ixxmp.IToken) *Node {
	var tokenList ixxmp.ITokenList

	if t, ok := list.(ixxmp.ITokenList); ok {
		tokenList = t
	}

	if tokenList == nil {
		return nil
	}
	offset := 0
	node := &Node{}
	items := tokenList.GetItems()
	size := len(items)

	if items[offset].GetTokenByte() == 0xf8 {

	}

	node.TagToken = items[offset]
	node.Tag = node.TagToken.GetTokenString()
	offset++

	attributeCount := (size - 2 + size%2) / 2
	for i := 0; i < attributeCount; i++ {
		k := items[offset].GetTokenString()
		offset++
		if items[offset] != nil {
			v := items[offset].GetTokenString()
			offset++
			node.Attributes = append(node.Attributes, NewAttribute(k, v))
		}
	}

	if size%2 == 1 {
		return node
	}

	child := items[offset]
	if child == nil {
		return node
	}
	if childTokenList, ok := child.(ixxmp.ITokenList); ok {
		for _, token := range childTokenList.GetItems() {
			node.Children = append(node.Children, n.From(token))
		}
	} else {
		node.Data = items[offset].GetTokenBytes()
	}
	return node
}
func (n *Node) GetTokenArray() ixxmp.ITokenList {
	defer func() {
		if r := recover(); r != nil {
			//打印错误堆栈信息
			log.Printf("GetTokenArray panic: %v\n", r)
		}
	}()
	size := 0
	if n == nil {
		return nil
	}
	if n.Data != nil {
		size++
	}
	if size <= 1 {
		attributes := n.Attributes
		attributeCount := len(attributes)
		if attributes != nil && len(attributes) != 0 {
			attributeCount = attributeCount << 1 // attributeCount * 2
		}
		size = attributeCount + 1 + size
		var tokenList ixxmp.ITokenList
		if size < 256 {
			tokenList = ShortArray()
		} else {
			tokenList = LongArray()
		}
		// write tag
		tokenList.AddItem(n.writeString(n.Tag, false))
		// write attributes
		if attributeCount > 0 {
			for _, attribute := range attributes {
				tokenList.AddItem(n.writeString(attribute.Key(), false))
				tokenList.AddItem(n.writeString(attribute.Value(), false))
			}
		}
		// write data
		if n.Data != nil && len(n.Data) != 0 {
			tokenList.AddItem(n.writeBytes(n.Data, false))
		} else if n.Children != nil && len(n.Children) > 0 {
			var childList ixxmp.ITokenList
			if len(n.Children) < 256 {
				childList = ShortArray()
			} else {
				childList = LongArray()
			}
			for _, child := range n.Children {
				childList.AddItem(child.GetTokenArray().(ixxmp.IToken))
			}
			tokenList.AddItem(childList.(ixxmp.IToken))
		}
		return tokenList
	}
	return nil
}
func (n *Node) writeBytes(d []byte, z bool) ixxmp.IToken {
	length := len(d)
	if length >= 256 {
		return Int24LengthArray(d)
	} else if length >= 1048576 {
		return Int32LengthArray(d)
	} else { // 数据小于 256位
		if z {
			nibble := PackedNibble(d)
			if nibble != nil {
				return nibble
			}
			h := PackedHex(d)
			if h != nil {
				return JoinToken(h, Int8LengthArray2(h.data, length))
			}
		}
		return Int8LengthArray(d)
	}
}
func (n *Node) writeString(s string, z bool) ixxmp.IToken {
	if len(dictionary) == 0 || len(secondaryDictionary) == 0 {
		panic(errors.New("dictionary or secondaryDictionary error"))
	}
	// dictionary
	for i := 0; i < len(dictionary); i++ {
		if strings.Compare(s, dictionary[i]) == 0 {
			return NewToken(byte(i))
		}
	}
	// secondaryDictionary
	for i := 0; i < len(secondaryDictionary); i++ {
		if strings.Compare(s, secondaryDictionary[i]) == 0 {
			return SecondaryToken(i)
		}
	}

	// jid
	if strings.Contains(s, "@") {
		jid := strings.Split(s, "@")
		user := n.writeString(jid[0], true)
		server := n.writeString(jid[1], false)
		return JabberId(user, server)
	}

	return n.writeBytes([]byte(s), z)
}
func (n *Node) GetString(index ...*int) string {
	builder := strings.Builder{}

	var (
		i    int
		tabs string
	)
	if len(index) > 0 {
		i = *index[0]
	}
	iout := i
	for iout > 0 {
		tabs += "    "
		iout--
	}
	i++
	builder.WriteString(tabs)
	builder.WriteString("<")
	if n.TagToken != nil && n.TagToken.GetTokenByte() == 2 {
		builder.WriteString("/")
	}
	builder.WriteString(n.Tag)
	for _, attribute := range n.Attributes {
		builder.WriteString(" ")
		builder.WriteString(attribute.Key())
		builder.WriteRune('=')
		builder.WriteRune('"')
		builder.WriteString(attribute.Value())
		builder.WriteRune('"')
	}

	if (n.Data == nil || len(n.Data) == 0) && len(n.Children) == 0 {
		if n.TagToken != nil && n.TagToken.GetTokenByte() == 1 {
			builder.WriteRune('>')
		} else {
			if n.TagToken != nil && n.TagToken.GetTokenByte() == 2 {
				builder.WriteRune('>')
			} else {
				builder.WriteString("/>")
			}
		}
	} else {
		builder.WriteRune('>')
		if n.Data == nil || len(n.Data) == 0 {
			builder.WriteRune('\n')

			for _, child := range n.Children {
				builder.WriteString(child.GetString(&i))
				builder.WriteRune('\n')
			}
		} else {
			dataBytes := n.Data
			builder.WriteString(hex.EncodeToString(dataBytes))
			//n.Data = dataBytes[2:]
		}
		if len(n.Children) != 0 {
			builder.WriteString(tabs)
		}
		builder.WriteString(fmt.Sprintf("</%s>", n.Tag))
	}

	/*inputReader := strings.NewReader(builder.String())
	p := xml.NewDecoder(inputReader)*/
	if i == 1 {
		return "\n" + builder.String()
	}

	return builder.String()
}
