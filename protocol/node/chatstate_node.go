package node

import "ws-go/protocol/newxxmp"

type ChatStateNode struct {
	*BaseNode
}

// createChatStatePaused 正在输入
func createChatStatePaused(to string) *ChatStateNode {
	/*
		<chatstate to="8617607567005-1619081389@g.us">
		    <paused/>
		</chatstate>
	*/
	c := &ChatStateNode{BaseNode: NewBaseNode()}
	chatStateNode := newxxmp.EmptyNode("chatstate")
	chatStateNode.Attributes.AddAttr("to", to)
	// paused node
	pausedNode := newxxmp.EmptyNode("paused")
	// set paused node to chat state node children
	chatStateNode.Children.AddNode(pausedNode)
	c.Node = chatStateNode

	return c
}

// createChatStateComposing 正在输入
func createChatStateComposing(to string) *ChatStateNode {
	/*
			<chatstate to="8617607567005-1619081389@g.us">
		    	<composing/>
			</chatstate>
	*/
	c := &ChatStateNode{BaseNode: NewBaseNode()}
	chatStateNode := newxxmp.EmptyNode("chatstate")
	chatStateNode.Attributes.AddAttr("to", to)
	// paused node
	pausedNode := newxxmp.EmptyNode("composing")
	// set paused node to chat state node children
	chatStateNode.Children.AddNode(pausedNode)
	c.Node = chatStateNode

	return c
}
