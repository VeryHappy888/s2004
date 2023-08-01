package xxmp

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func NewNode(tag string, attrs []Attribute, children []*Node, data []byte) *Node {
	return &Node{
		TagToken:   nil,
		Tag:        tag,
		Attributes: attrs,
		Children:   children,
		Data:       data,
	}
}

type Node struct {
	TagToken   tokenInterface
	Tag        string
	Attributes []Attribute
	Children   []*Node
	Data       []byte

	index int
}

func (n *Node) From(list AbstractListInterface) *Node {
	offset := 0
	node := &Node{}
	items := list.GetItems()
	size := len(items)

	node.TagToken = items[offset]
	node.Tag = node.TagToken.getString()
	offset++

	attributeCount := (size - 2 + size%2) / 2
	for i := 0; i < attributeCount; i++ {
		k := items[offset].getString()
		offset++
		v := items[offset].getString()
		offset++
		node.Attributes = append(node.Attributes, NewAttribute(k, v))
	}

	if size%2 == 1 {
		return node
	}

	child := items[offset]
	listInterface, ok := child.(AbstractListInterface)
	if ok {
		for _, t := range listInterface.GetItems() {
			node.Children = append(node.Children, node.From(t.(AbstractListInterface)))
		}
	} else {
		node.Data = child.getBytes()
	}
	return node
}

func (n *Node) GetToken() AbstractListInterface {
	size := 1
	size += len(n.Attributes) * 2

	if len(n.Data) != 0 || len(n.Children) > 0 {
		size++
	}

	var nodeList AbstractListInterface

	if size >= 256 {
		nodeList = LongList()
	} else {
		nodeList = ShortList()
	}

	if n.TagToken == nil {
		nodeList.AddItem(n.writeString(n.Tag))
	} else {
		nodeList.AddItem(n.TagToken)
	}

	// Attributes
	for _, attribute := range n.Attributes {
		nodeList.AddItem(n.writeString(attribute.Key()))
		nodeList.AddItem(n.writeString(attribute.Value()))
	}

	if len(n.Children) > 0 {
		var childList AbstractListInterface
		if len(n.Children) >= 256 {
			childList = LongList()
		} else {
			childList = ShortList()
		}

		for _, child := range n.Children {
			childList.AddItems(child.GetToken())
		}
		nodeList.AddItem(childList.(tokenInterface))
	} else if n.Data != nil && len(n.Data) > 0 {
		nodeList.AddItem(n.writeString(string(n.Data)))
	}

	return nodeList
}

// writeString
func (n *Node) writeString(s string) tokenInterface {
	if len(s) == 0 {
		return NewXMMPToken(0)
	}

	// dictionary
	for i := 0; i < len(dictionary); i++ {
		if strings.Compare(s, dictionary[i]) == 0 {
			return NewXMMPToken(byte(i))
		}
	}

	// secondaryDictionary
	for i := 0; i < len(secondaryDictionary); i++ {
		if strings.Compare(s, secondaryDictionary[i]) == 0 {
			return NewSecondaryToken(i)
		}
	}

	// jid
	if strings.Contains(s, "@") {
		jid := strings.Split(s, "@")
		user := n.writeString(jid[0])
		server := n.writeString(jid[1])
		return NewJabberId(user, server)
	}

	if len(s) <= 255 {
		return NewInt8LengthArrayString(s)
	}

	if len(s) <= 1048575 {
		return NewInt8LengthArrayString(s)
	}

	if len(s) <= 2147483647 {
		return NewInt31LengthArrayString(s)
	}

	return nil
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
	if n.TagToken.getToken() == 2 {
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

	if n.Data == nil && len(n.Children) == 0 {
		if n.TagToken.getToken() == 1 {
			builder.WriteRune('>')
		} else {
			if n.TagToken.getToken() == 2 {
				builder.WriteRune('>')
			} else {
				builder.WriteString("/>")
			}
		}
	} else {
		builder.WriteRune('>')
		if n.Data == nil {
			builder.WriteRune('\n')
			for _, child := range n.Children {
				builder.WriteString(child.GetString(&i))
				builder.WriteRune('\n')
			}
		} else {
			dataBytes := []byte(n.Data)
			builder.WriteString(hex.EncodeToString(dataBytes))
		}
		if len(n.Children) != 0 {
			builder.WriteString(tabs)
		}
		builder.WriteString(fmt.Sprintf("</%s>", n.Tag))
	}

	/*inputReader := strings.NewReader(builder.String())
	p := xml.NewDecoder(inputReader)*/

	return builder.String()
}
