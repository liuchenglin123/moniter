package async

import (
	"errors"
	"fmt"
	"time"
)

// Task is a struct that represents a task
type Task struct {
	name        string              // task的名称
	collectTime time.Duration       // 收集时间
	channelLen  int64               // 临时chan长度
	collect     chan interface{}    // 全部收集的信息
	data        []interface{}       // 部分收集的信息
	maxDataLen  int                 // 临时信息最大长度，超过后会触发consumer
	consumer    func([]interface{}) // 消费函数 接收一个已经关闭的chan

	close chan chan struct{} // 关闭任务
}

func CPUTask() *Task {
	name := "cpu"

	t, ok := tasks[name]
	if ok {
		return t
	}
	tasks[name] = newTask(name, int64(channelLen*5), time.Second*5, channelLen)
	return tasks[name]
}
func MemoryTask() *Task {
	name := "memory"

	t, ok := tasks[name]
	if ok {
		return t
	}
	tasks[name] = newTask(name, int64(channelLen*5), time.Second*5, channelLen)
	return tasks[name]
}
func IOTask() *Task {
	name := "io"

	t, ok := tasks[name]
	if ok {
		return t
	}
	tasks[name] = newTask(name, int64(channelLen*5), time.Second*5, channelLen)
	return tasks[name]
}
func newTask(name string, channelLen int64, collectTime time.Duration, maxDataLen int) *Task {
	return &Task{
		name:        name,
		collectTime: collectTime,
		channelLen:  channelLen,
		collect:     make(chan interface{}, channelLen),
		data:        make([]interface{}, 0),
		maxDataLen:  maxDataLen,
		close:       make(chan chan struct{}, 1),
	}
}

// SetConsumer 设置消费函数
func (t *Task) SetConsumer(consumer func([]interface{})) {
	t.consumer = consumer
}

// Async 异步执行
func (t *Task) Async() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("recover Async:", err)
			}
		}()

		t1 := time.Tick(t.collectTime)
		for {
			select {
			case v := <-t.collect:
				t.data = append(t.data, v)
				if t.maxDataLen > 0 && len(t.data) >= t.maxDataLen {
					tt := t.data
					t.data = make([]interface{}, 0, len(tt))
					if len(tt) > 0 {
						t.callConsumer(tt)
					}
				}
			case <-t1:
				// ** 不能判断 len(t.data)==0，有的任务需要依赖当前任务的定时执行
				tt := t.data
				t.data = make([]interface{}, 0, len(tt)/2)
				if len(tt) > 0 {
					t.callConsumer(tt)
				}
			case cc := <-t.close:
				// 收到关机信号后，关闭chan，其后产生的消息放弃处理
				close(t.collect)
				for v := range t.collect {
					t.data = append(t.data, v)
				}
				if len(t.data) > 0 {
					t.callConsumer(t.data)
				}
				cc <- struct{}{}
				close(cc)
				return
			}
		}
	}()
}

// Pub 尽量不要推指针数据进来
func (t *Task) Pub(b interface{}) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			fmt.Println("recover pub:", rec)
			err = errors.New("recover")
		}
	}()
	t.collect <- b
	return nil
}

func (t *Task) callConsumer(data []interface{}) {
	// 不要判断len(data)
	defer func() {
		if err := recover(); err != nil {
		}
	}()

	t.consumer(data)
}
