// Package buyback 是回购队列：每人槽位、过期清理、价格快照与回购幂等。
// 缺陷：每人槽位不生效、过期不清理、回购不幂等、回购价格取最新而不是最早快照、
// 每次查询扫描全部记录。
package buyback

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
	items   []entry
	memo    map[string]bool
	scanned int
}

// New 建回购队列。
func New(slots int, ttlMs int64) *Queue {
	return &Queue{slots: slots, ttlMs: ttlMs, memo: map[string]bool{}}
}

// Sell 卖出一件物品，进入回购队列。
func (q *Queue) Sell(player, item string, price int, nowMs int64) {
	q.scanned += len(q.items)
	q.items = append(q.items, entry{player: player, item: item, price: price, at: nowMs})
}

// Buy 回购一件物品，返回价格快照。
func (q *Queue) Buy(player, item string, nowMs int64, reqID string) (int, bool) {
	q.scanned += len(q.items)
	q.memo[reqID] = true
	price := 0
	found := false
	for _, rec := range q.items {
		if rec.player == player && rec.item == item {
			price = rec.price
			found = true
		}
	}
	return price, found
}

// List 返回该玩家当前可回购的物品。
func (q *Queue) List(player string, nowMs int64) []string {
	q.scanned += len(q.items)
	out := []string{}
	for _, rec := range q.items {
		if rec.player == player {
			out = append(out, rec.item)
		}
	}
	return out
}

// Scanned 是累计扫描量（规模观测）。
func (q *Queue) Scanned() int { return q.scanned }
