// Package buyback 是回购队列：每人槽位、过期清理、价格快照与回购幂等。
// 记录按玩家分桶存放，Sell、List 与 Buy 只触碰该玩家的记录，
// 单次代价摊还 O(命中记录数)，不随总记录数线性增长。
package buyback

import "sort"

type entry struct {
	player string
	item   string
	price  int
	at     int64
}

// Queue 是回购队列。
type Queue struct {
	slots   int
	ttlMs   int64
	byUser  map[string][]entry
	memo    map[string]bool
	scanned int
}

// New 建回购队列。
func New(slots int, ttlMs int64) *Queue {
	return &Queue{slots: slots, ttlMs: ttlMs, byUser: map[string][]entry{}, memo: map[string]bool{}}
}

// expired 判断记录在 nowMs 时刻是否已失效：存在时间达到 ttlMs 即失效。
func (q *Queue) expired(e entry, nowMs int64) bool {
	return nowMs-e.at >= q.ttlMs
}

// Sell 卖出一件物品，进入回购队列。
// 同一玩家最多保留 slots 条记录，超出时移除该玩家最早的一条。
func (q *Queue) Sell(player, item string, price int, nowMs int64) {
	entries := q.byUser[player]
	kept := entries[:0]
	for _, e := range entries {
		q.scanned++
		if !q.expired(e, nowMs) {
			kept = append(kept, e)
		}
	}
	entries = append(kept, entry{player: player, item: item, price: price, at: nowMs})
	if len(entries) > q.slots {
		oldest := 0
		for i := 1; i < len(entries); i++ {
			q.scanned++
			if entries[i].at < entries[oldest].at {
				oldest = i
			}
		}
		entries = append(entries[:oldest], entries[oldest+1:]...)
	}
	q.byUser[player] = entries
}

// Buy 回购一件物品，返回最早那条卖出记录的价格快照。
// 同一个 reqID 只成功一次，重复调用返回 (0, false)。
func (q *Queue) Buy(player, item string, nowMs int64, reqID string) (int, bool) {
	if q.memo[reqID] {
		return 0, false
	}
	entries := q.byUser[player]
	match := -1
	for i, e := range entries {
		q.scanned++
		if q.expired(e, nowMs) || e.item != item {
			continue
		}
		if match == -1 || e.at < entries[match].at {
			match = i
		}
	}
	if match == -1 {
		return 0, false
	}
	price := entries[match].price
	kept := entries[:0]
	for i, e := range entries {
		if i == match || q.expired(e, nowMs) {
			continue
		}
		kept = append(kept, e)
	}
	q.byUser[player] = kept
	q.memo[reqID] = true
	return price, true
}

// List 返回该玩家当前可回购的物品，按卖出时间升序。
func (q *Queue) List(player string, nowMs int64) []string {
	entries := q.byUser[player]
	hits := make([]entry, 0, len(entries))
	for _, e := range entries {
		q.scanned++
		if !q.expired(e, nowMs) {
			hits = append(hits, e)
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].at < hits[j].at })
	out := make([]string, 0, len(hits))
	for _, e := range hits {
		out = append(out, e.item)
	}
	return out
}

// Scanned 是累计扫描量（规模观测）。
func (q *Queue) Scanned() int { return q.scanned }
