package entity

type Receipt struct {
	RecipientId string
	MsgId       string
	ReceiptType string
	Participant string
}

func NewReceipt(recipientId, msgId, eType, p string) *Receipt {
	return &Receipt{
		RecipientId: recipientId,
		MsgId:       msgId,
		ReceiptType: eType,
		Participant: p,
	}
}
