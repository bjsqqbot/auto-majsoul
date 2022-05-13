package middleware

type MessageQueue struct {
	SelfDrawCh chan SelfDrawMessage
	MeldCh     chan MeldMessage
}

type SelfDrawMessage struct {
	HandTile34 []int
	TileGot    int
	BestCard   int
	Reach      bool
}

type MeldMessage struct {
	HandTile34 []int
	TileGot    int
	BestCard   int
	Chi        bool
	Pong       bool
	Kan        bool
	Long       bool
}

var MQ MessageQueue

func init() {
	MQ = MessageQueue{
		SelfDrawCh: make(chan SelfDrawMessage),
		MeldCh:     make(chan MeldMessage),
	}
}

func (mq *MessageQueue) Send(m interface{}) {
	switch m := m.(type) {
	case SelfDrawMessage:
		mq.SelfDrawCh <- m
	case MeldMessage:
		mq.MeldCh <- m
	default:
	}
}
