package buyback

import (
	"fmt"
	"testing"
)

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

func TestSlotsEvictOldest(t *testing.T) {
	queue := New(2, 100000)
	queue.Sell("p1", "item-0", 10, 0)
	queue.Sell("p1", "item-1", 10, 1)
	queue.Sell("p1", "item-2", 10, 2)
	got := queue.List("p1", 10)
	if len(got) != 2 || got[0] != "item-1" || got[1] != "item-2" {
		t.Fatalf("每人只应保留 slots 条且移除最早的一条，实际 %v", got)
	}
}

func TestSlotsArePerPlayer(t *testing.T) {
	queue := New(2, 100000)
	queue.Sell("p1", "a", 10, 0)
	queue.Sell("p1", "b", 10, 1)
	queue.Sell("p2", "c", 10, 2)
	if got := queue.List("p2", 10); len(got) != 1 || got[0] != "c" {
		t.Fatalf("槽位应该按玩家计算，实际 %v", got)
	}
	if got := queue.List("p1", 10); len(got) != 2 {
		t.Fatalf("别的玩家不该占用 p1 的槽位，实际 %v", got)
	}
}

func TestTTLExpiry(t *testing.T) {
	queue := New(4, 1000)
	queue.Sell("p1", "sword", 100, 0)
	if got := queue.List("p1", 999); len(got) != 1 {
		t.Fatalf("ttl 内仍应可见，实际 %v", got)
	}
	if got := queue.List("p1", 1000); len(got) != 0 {
		t.Fatalf("存在时间达到 ttlMs 后应失效，实际 %v", got)
	}
	if _, ok := queue.Buy("p1", "sword", 1001, "r1"); ok {
		t.Fatal("过期记录不应能被 Buy 回购")
	}
}

func TestBuyPriceSnapshot(t *testing.T) {
	queue := New(4, 100000)
	queue.Sell("p1", "sword", 100, 0)
	queue.Sell("p1", "sword", 300, 10)
	price, ok := queue.Buy("p1", "sword", 20, "r1")
	if !ok || price != 100 {
		t.Fatalf("Buy 应返回最早卖出记录的价格快照，实际 (%d, %v)", price, ok)
	}
	price, ok = queue.Buy("p1", "sword", 20, "r2")
	if !ok || price != 300 {
		t.Fatalf("最早一条回购后应轮到下一条，实际 (%d, %v)", price, ok)
	}
	if got := queue.List("p1", 20); len(got) != 0 {
		t.Fatalf("回购成功后记录应移除，实际 %v", got)
	}
}

func TestBuyIdempotent(t *testing.T) {
	queue := New(4, 100000)
	queue.Sell("p1", "sword", 100, 0)
	if _, ok := queue.Buy("p1", "sword", 0, "same"); !ok {
		t.Fatal("第一次回购应该成功")
	}
	price, ok := queue.Buy("p1", "sword", 0, "same")
	if ok || price != 0 {
		t.Fatalf("同一 reqID 重复调用应返回 (0, false)，实际 (%d, %v)", price, ok)
	}
}

func TestListOrderedBySellTime(t *testing.T) {
	queue := New(4, 100000)
	queue.Sell("p1", "late", 10, 20)
	queue.Sell("p1", "early", 10, 10)
	got := queue.List("p1", 30)
	if len(got) != 2 || got[0] != "early" || got[1] != "late" {
		t.Fatalf("List 应按卖出时间升序，实际 %v", got)
	}
}

func TestScannedDoesNotGrowWithRecords(t *testing.T) {
	queue := New(10000, 100000)
	for index := 0; index < 3000; index++ {
		queue.Sell(fmt.Sprintf("p-%d", index), "sword", 100, int64(index))
	}
	before := queue.Scanned()
	for index := 0; index < 3000; index++ {
		queue.List("p-2999", 500000)
	}
	if delta := queue.Scanned() - before; delta > 6000 {
		t.Fatalf("List 扫描量不应随记录数线性增长，3000 次查询扫描了 %d", delta)
	}
}
