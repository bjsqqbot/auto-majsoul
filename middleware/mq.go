package middleware

type MessageQueue struct {
	ch chan TileMessage
}

type TileMessage struct {
	HandTile34 []int
	TileGot    int
	BestCard   int
	Chi        bool
	Peng       bool
	Gang       bool
	Reach      bool
}

var MQ MessageQueue = MessageQueue{ch: make(chan TileMessage)}

func (mq *MessageQueue) Send(m TileMessage) {
	mq.ch <- m
}

func (mq *MessageQueue) Receive() TileMessage {
	m := <-mq.ch
	return m
}
