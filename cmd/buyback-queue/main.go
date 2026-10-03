// Command buyback-queue 跑回购样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"example.com/buyback"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("buyback-queue", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "slots", "slots / ttl / idem / price / scan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	switch *sample {
	case "slots":
		queue := buyback.New(2, 100000)
		for index := 0; index < 3; index++ {
			queue.Sell("p1", fmt.Sprintf("item-%d", index), 10, int64(index))
		}
		fmt.Fprintf(stdout, "listed=%d\n", len(queue.List("p1", 10)))
	case "ttl":
		queue := buyback.New(4, 1000)
		queue.Sell("p1", "sword", 100, 0)
		fmt.Fprintf(stdout, "listed=%d\n", len(queue.List("p1", 1001)))
	case "idem":
		queue := buyback.New(4, 100000)
		queue.Sell("p1", "sword", 100, 0)
		queue.Buy("p1", "sword", 0, "same")
		_, ok := queue.Buy("p1", "sword", 0, "same")
		fmt.Fprintf(stdout, "second=%v\n", ok)
	case "price":
		queue := buyback.New(4, 100000)
		queue.Sell("p1", "sword", 100, 0)
		queue.Sell("p1", "sword", 300, 10)
		price, _ := queue.Buy("p1", "sword", 20, "r1")
		fmt.Fprintf(stdout, "price=%d\n", price)
	case "scan":
		queue := buyback.New(10000, 100000)
		for index := 0; index < 3000; index++ {
			queue.Sell(fmt.Sprintf("p-%d", index), "sword", 100, int64(index))
		}
		for index := 0; index < 3000; index++ {
			queue.List("p-2999", 0)
		}
		fmt.Fprintf(stdout, "scanned=%d\n", queue.Scanned())
	default:
		fmt.Fprintln(stderr, "需要 --sample slots|ttl|idem|price|scan")
		return 2
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
