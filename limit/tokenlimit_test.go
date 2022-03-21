package limit_test

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gozerosource/limit/core/limit"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

func Test_TokenLimiter(t *testing.T) {
	const (
		burst   = 100
		rate    = 100
		seconds = 5
	)
	store := redis.New("127.0.0.1:6379")
	fmt.Println(store.Ping())
	// New tokenLimiter
	limiter := limit.NewTokenLimiter(rate, burst, store, "token-limiter")
	timer := time.NewTimer(time.Second * seconds)
	quit := make(chan struct{})
	defer timer.Stop()
	go func() {
		<-timer.C
		close(quit)
	}()

	var allowed, denied int32
	var wait sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wait.Add(1)
		go func() {
			for {
				select {
				case <-quit:
					wait.Done()
					return
				default:
					if limiter.Allow() {
						atomic.AddInt32(&allowed, 1)
					} else {
						atomic.AddInt32(&denied, 1)
					}
				}
			}
		}()
	}

	wait.Wait()
	fmt.Printf("allowed: %d, denied: %d, qps: %d\n", allowed, denied, (allowed+denied)/seconds)
}
