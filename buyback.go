// Package buyback 是回购队列：每人槽位、过期清理、价格快照与回购幂等。
package buyback

import "container/list"

type entry struct {
	player string
	item   string
	price  int
	at     int64
	rowEl  *list.Element
	itemEl *list.Element
}

type playerQueue struct {
	rows   *list.List
	byItem map[string]*list.List
}

// Queue 是回购队列。
type Queue struct {
	slots   int
	ttlMs   int64
	players map[string]*playerQueue
	memo    map[string]bool
	scanned int
}

// New 建回购队列。
func New(slots int, ttlMs int64) *Queue {
	return &Queue{
		slots:   slots,
		ttlMs:   ttlMs,
		players: map[string]*playerQueue{},
		memo:    map[string]bool{},
	}
}

func (q *Queue) player(player string) *playerQueue {
	pq := q.players[player]
	if pq == nil {
		pq = &playerQueue{rows: list.New(), byItem: map[string]*list.List{}}
		q.players[player] = pq
	}
	return pq
}

func (q *Queue) expired(rec *entry, nowMs int64) bool {
	return nowMs-rec.at >= q.ttlMs
}

// remove 从玩家链表与物品链表中同时摘除一条记录。
func (pq *playerQueue) remove(rec *entry) {
	pq.rows.Remove(rec.rowEl)
	items := pq.byItem[rec.item]
	if items != nil {
		items.Remove(rec.itemEl)
		if items.Len() == 0 {
			delete(pq.byItem, rec.item)
		}
	}
}

// Sell 卖出一件物品，进入回购队列。
func (q *Queue) Sell(player, item string, price int, nowMs int64) {
	pq := q.player(player)
	rec := &entry{player: player, item: item, price: price, at: nowMs}
	rec.rowEl = pq.rows.PushBack(rec)
	items := pq.byItem[item]
	if items == nil {
		items = list.New()
		pq.byItem[item] = items
	}
	rec.itemEl = items.PushBack(rec)
	if pq.rows.Len() > q.slots {
		pq.remove(pq.rows.Front().Value.(*entry))
	}
}

// Buy 回购一件物品，返回最早那条卖出记录的价格快照。
func (q *Queue) Buy(player, item string, nowMs int64, reqID string) (int, bool) {
	if q.memo[reqID] {
		return 0, false
	}
	q.memo[reqID] = true

	pq := q.players[player]
	if pq == nil {
		return 0, false
	}
	items := pq.byItem[item]
	if items == nil {
		return 0, false
	}
	for items.Len() > 0 {
		rec := items.Front().Value.(*entry)
		q.scanned++
		if !q.expired(rec, nowMs) {
			price := rec.price
			pq.remove(rec)
			return price, true
		}
		pq.remove(rec)
	}
	return 0, false
}

// List 返回该玩家当前可回购的物品，按卖出时间升序。
func (q *Queue) List(player string, nowMs int64) []string {
	out := []string{}
	pq := q.players[player]
	if pq == nil {
		return out
	}
	for el := pq.rows.Front(); el != nil; {
		rec := el.Value.(*entry)
		next := el.Next()
		q.scanned++
		if q.expired(rec, nowMs) {
			pq.remove(rec)
		} else {
			out = append(out, rec.item)
		}
		el = next
	}
	return out
}

// Scanned 是累计扫描量（规模观测）。
func (q *Queue) Scanned() int { return q.scanned }
