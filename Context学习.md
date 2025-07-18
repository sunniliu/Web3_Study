# Context学习
## 基本概念
在 Go 语言中，context 包用于在多个 goroutine 之间传递请求范围的数据、取消信号、超时或截止时间

## 基本用法
### 1.传递数据
```
package main

import (
	"context"
	"fmt"
)

func main() {
	// 创建根上下文
	ctx := context.Background()

	// 附加键值对数据（key 建议用自定义类型避免冲突）
	type key string
	userIDKey := key("user_id")
	ctx = context.WithValue(ctx, userIDKey, "123")

	// 在函数中读取数据
	printUserID(ctx, userIDKey)
}

func printUserID(ctx context.Context, key key) {
	if v := ctx.Value(key); v != nil {
		fmt.Printf("User ID: %v\n", v) // 输出: User ID: 123
	}
}
```

### 2.超时控制
```
package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// 设置 2 秒超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel() // 防止资源泄漏

	// 模拟耗时操作
	select {
	case <-time.After(3 * time.Second):
		fmt.Println("操作完成")
	case <-ctx.Done():
		fmt.Println("操作超时:", ctx.Err()) // 输出: 操作超时: context deadline exceeded
	}
}
```

### 3.手动取消
```
package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动一个 goroutine 执行任务
	go func() {
		select {
		case <-time.After(5 * time.Second):
			fmt.Println("任务完成")
		case <-ctx.Done():
			fmt.Println("任务取消:", ctx.Err())
		}
	}()

	// 模拟用户 2 秒后取消
	time.Sleep(2 * time.Second)
	cancel() // 触发取消

	time.Sleep(1 * time.Second) // 等待输出
}
// 输出: 任务取消: context canceled
```

### 级联取消
```
package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// 父上下文（带超时）
	parentCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 子上下文（继承父的取消逻辑）
	childCtx := context.WithValue(parentCtx, "requestID", "abc123")

	// 监听取消信号
	go handleRequest(childCtx)

	// 等待父上下文超时
	<-parentCtx.Done()
	fmt.Println("主函数退出:", parentCtx.Err())
}

func handleRequest(ctx context.Context) {
	select {
	case <-time.After(5 * time.Second):
		fmt.Println("处理完成, Request ID:", ctx.Value("requestID"))
	case <-ctx.Done():
		fmt.Println("请求取消:", ctx.Err()) // 输出: 请求取消: context deadline exceeded
	}
}   
```

## 源码分析
数据结构:
```
type Context interface {
    Deadline() (deadline time.Time, ok bool) // 返回超时时间（如果有）
    Done() <-chan struct{}                  // 返回取消信号通道
    Err() error                             // 返回取消原因
    Value(key any) any                      // 获取上下文值
}
```
从数据结构可以看出，context承载了值传递、取消任务、超时任务的处理
设计的十分简洁

重点研究了下cancel和done两个函数，主要对这2个函数的源码进行分析
总共四个步骤
1. 检查是否已取消	如果已经取消过，直接返回（避免重复操作）
2. 关闭 done 通道	调用 close(done)
3. 通知所有子任务	遍历 children，逐个调用它们的 cancel()
4. 清理自己	从父任务的 children 中移除自己（避免内存泄漏

源码如下
```
func (c *cancelCtx) cancel(removeFromParent bool, err error) {
    // 1. 检查是否已取消（避免重复操作）
    c.mu.Lock()
    if c.err != nil {
        c.mu.Unlock()
        return // 已经取消过，直接返回
    }
    c.err = err // 记录取消原因（Canceled 或 DeadlineExceeded）

    // 2. 关闭 done 通道（关键步骤！）
    if c.done == nil {
        c.done = closedchan // 预定义的已关闭通道（优化）
    } else {
        close(c.done) // 正常关闭通道
    }

    // 3. 级联取消所有子 context
    for child := range c.children {
        child.cancel(false, err) // 递归取消子节点
    }
    c.children = nil // 清空子节点map
    c.mu.Unlock()

    // 4. 从父context中移除自己（避免内存泄漏）
    if removeFromParent {
        removeChild(c.Context, c)
    }
}
```

最重要的是关闭了done通道，goroutine 会 非阻塞地收到一个零值（struct{}{}），从而去做一些如资源释放，消息通知的事情

done函数通过懒加载的方式去实现

```
func (c *cancelCtx) Done() <-chan struct{} {
    d := c.done.Load()
    if d != nil {
        return d.(chan struct{})
    }
    c.mu.Lock()
    defer c.mu.Unlock()
    d = c.done.Load()
    if d == nil {
        d = make(chan struct{})
        c.done.Store(d)
    }
    return d.(chan struct{})
}

```

WithDeadline源码分析

```

func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
    return WithDeadline(parent, time.Now().Add(timeout))
}

func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
    // 检查父 Context 是否已有更早的截止时间
    if cur, ok := parent.Deadline(); ok && cur.Before(d) {
        return WithCancel(parent)
    }
    c := &timerCtx{
        cancelCtx: newCancelCtx(parent),
        deadline:  d,
    }
    // 设置定时器自动取消
    c.timer = time.AfterFunc(time.Until(d), func() {
        c.cancel(true, DeadlineExceeded)
    })
    return c, func() { c.cancel(true, Canceled) }
}

```

通过定时器，注册回调函数，到截止时间后，执行cancel函数


