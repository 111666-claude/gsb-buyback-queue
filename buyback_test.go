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

func TestSlotsEvictsOldest(t *testing.T) {
	queue := New(2, 100000)
	queue.Sell("p1", "a", 10, 0)
	queue.Sell("p1", "b", 10, 1)
	queue.Sell("p1", "c", 10, 2)
	got := queue.List("p1", 3)
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("超出槽位应该移除最早的一条，得到 %v", got)
	}
}

func TestSlotsArePerPlayer(t *testing.T) {
	queue := New(1, 100000)
	queue.Sell("p1", "a", 10, 0)
	queue.Sell("p2", "b", 10, 0)
	queue.Sell("p1", "c", 10, 1)
	if got := queue.List("p2", 2); len(got) != 1 || got[0] != "b" {
		t.Fatalf("p2 的记录不该被 p1 挤掉，得到 %v", got)
	}
	if got := queue.List("p1", 2); len(got) != 1 || got[0] != "c" {
		t.Fatalf("p1 应该只剩最新一条，得到 %v", got)
	}
}

func TestTTLExpiryList(t *testing.T) {
	queue := New(4, 1000)
	queue.Sell("p1", "sword", 100, 0)
	if got := queue.List("p1", 999); len(got) != 1 {
		t.Fatalf("未到 ttl 应该还能看到，得到 %v", got)
	}
	if got := queue.List("p1", 1000); len(got) != 0 {
		t.Fatalf("达到 ttl 应该失效，得到 %v", got)
	}
}

func TestTTLExpiryBuy(t *testing.T) {
	queue := New(4, 1000)
	queue.Sell("p1", "sword", 100, 0)
	if _, ok := queue.Buy("p1", "sword", 1000, "r1"); ok {
		t.Fatal("过期记录不该能回购")
	}
}

func TestPriceSnapshotUsesEarliest(t *testing.T) {
	queue := New(4, 100000)
	queue.Sell("p1", "sword", 100, 0)
	queue.Sell("p1", "sword", 300, 10)
	price, ok := queue.Buy("p1", "sword", 20, "r1")
	if !ok || price != 100 {
		t.Fatalf("回购应该返回最早快照价 100，得到 price=%d ok=%v", price, ok)
	}
	price, ok = queue.Buy("p1", "sword", 20, "r2")
	if !ok || price != 300 {
		t.Fatalf("第二次回购应该轮到 300 那条，得到 price=%d ok=%v", price, ok)
	}
}

func TestBuyIdempotent(t *testing.T) {
	queue := New(4, 100000)
	queue.Sell("p1", "sword", 100, 0)
	price, ok := queue.Buy("p1", "sword", 0, "same")
	if !ok || price != 100 {
		t.Fatalf("第一次回购应该成功，得到 price=%d ok=%v", price, ok)
	}
	price, ok = queue.Buy("p1", "sword", 0, "same")
	if ok || price != 0 {
		t.Fatalf("同一 reqID 重复回购应该返回 (0, false)，得到 price=%d ok=%v", price, ok)
	}
}

func TestBuyRemovesRecord(t *testing.T) {
	queue := New(4, 100000)
	queue.Sell("p1", "sword", 100, 0)
	queue.Buy("p1", "sword", 0, "r1")
	if got := queue.List("p1", 0); len(got) != 0 {
		t.Fatalf("回购成功后记录应该移除，得到 %v", got)
	}
	if _, ok := queue.Buy("p1", "sword", 0, "r2"); ok {
		t.Fatal("记录已移除，换个 reqID 也不该再买到")
	}
}

func TestListOrderAscending(t *testing.T) {
	queue := New(4, 100000)
	queue.Sell("p1", "a", 10, 0)
	queue.Sell("p1", "b", 10, 5)
	queue.Sell("p1", "c", 10, 9)
	got := queue.List("p1", 10)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("List 应该按卖出时间升序，得到 %v", got)
	}
}

func TestScannedNotLinearInRecords(t *testing.T) {
	queue := New(10000, 100000)
	for index := 0; index < 3000; index++ {
		queue.Sell(fmt.Sprintf("p-%d", index), "sword", 100, int64(index))
	}
	before := queue.Scanned()
	for index := 0; index < 3000; index++ {
		queue.List("p-empty", 0)
	}
	if delta := queue.Scanned() - before; delta != 0 {
		t.Fatalf("查不到记录的 List 不该扫描，多扫了 %d 条", delta)
	}
	for index := 0; index < 3000; index++ {
		queue.List("p-2999", 0)
	}
	if scanned := queue.Scanned(); scanned > 6000 {
		t.Fatalf("3000 次查询每次只该命中 1 条，Scanned 得到 %d", scanned)
	}
}

func TestRepeatSequenceDeterministic(t *testing.T) {
	run := func() ([]string, int, bool) {
		queue := New(2, 1000)
		queue.Sell("p1", "a", 10, 0)
		queue.Sell("p1", "b", 20, 1)
		queue.Sell("p1", "c", 30, 2)
		price, ok := queue.Buy("p1", "b", 3, "r1")
		return queue.List("p1", 3), price, ok
	}
	list1, price1, ok1 := run()
	list2, price2, ok2 := run()
	if price1 != price2 || ok1 != ok2 || len(list1) != len(list2) {
		t.Fatal("同一串操作重复执行结果应该相同")
	}
	for index := range list1 {
		if list1[index] != list2[index] {
			t.Fatal("同一串操作重复执行结果应该相同")
		}
	}
}
