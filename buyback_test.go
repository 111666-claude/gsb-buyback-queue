package buyback

import "testing"

func TestEmptyList(t *testing.T) {
	if len(New(4, 1000).List("p1", 0)) != 0 {
		t.Fatal("没有卖出记录时不该有可回购物品")
	}
}

func TestSellThenList(t *testing.T) {
	queue := New(4, 1000)
	queue.Sell("p1", "sword", 100, 0)
	if len(queue.List("p1", 0)) != 1 {
		t.Fatal("卖出之后应该有一条记录")
	}
}

func TestBuyUnknownItem(t *testing.T) {
	queue := New(4, 1000)
	if _, ok := queue.Buy("p1", "ghost", 0, "r1"); ok {
		t.Fatal("没有卖出过的物品不该能回购")
	}
}

func TestSlotsStored(t *testing.T) {
	if New(3, 1000).slots != 3 {
		t.Fatal("槽位数应该能读回来")
	}
}

func TestScannedStartsAtZero(t *testing.T) {
	if New(4, 1000).Scanned() != 0 {
		t.Fatal("Scanned 初始应该是 0")
	}
}
