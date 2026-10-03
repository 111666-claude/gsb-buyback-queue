# buyback-queue

回购队列：每人槽位、过期清理、价格快照与回购幂等，只用 Go 标准库。

```
go run ./cmd/buyback-queue --sample=slots
go run ./cmd/buyback-queue --sample=ttl
go run ./cmd/buyback-queue --sample=idem
go run ./cmd/buyback-queue --sample=price
go run ./cmd/buyback-queue --sample=scan
go test ./...
go vet ./...
```

## 口径（README 为准）

- **每人槽位**：同一个玩家最多保留 `slots` 条回购记录，超出时移除该玩家最早的一条。
- **过期**：卖出记录存在时间达到 `ttlMs` 后失效，`List` 与 `Buy` 都不再看到它。
- **价格快照**：`Buy` 返回的是这件物品最早那条卖出记录的价格，不受后续价格影响。
- **回购幂等**：同一个 `reqID` 只成功一次，重复调用返回 `(0, false)`。
- **顺序**：`List` 按卖出时间升序返回物品名。
- **不变量**：`List` 条数不超过 `slots`；回购成功后该记录从队列移除；
  同一串卖出与回购序列重复执行结果相同。
- **代价**：`List` 与 `Buy` 不许扫描全部记录，`Scanned` 不随记录数线性增长。
- **规模**：10 万条记录、每秒 10 万次查询，单次摊还 O(命中记录数)。

## 输出契约（不改格式）

```
listed=..
second=..
price=..
scanned=..
```

## 目录

```
buyback.go              槽位、过期、快照与幂等
buyback_test.go         单元测试
cmd/buyback-queue/      命令行入口
```
