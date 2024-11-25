package async

import (
	"fmt"
	"sync"
	"time"
)

var tasks = make(map[string]*Task)

// 基础支持 qps 1024
const channelLen int = 1024

// ShutDown 终止
func ShutDown() {
	var wg sync.WaitGroup
	for _, t := range tasks {
		wg.Add(1)
		go func(t *Task) {
			time.Sleep(time.Second)
			cc := make(chan struct{}, 1)
			fmt.Println("to close ", t.name)

			t.close <- cc
			<-cc
			fmt.Println("closed ", t.name)
			wg.Done()
		}(t)
	}
	wg.Wait()
}
