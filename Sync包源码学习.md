# Sync源码学习

## 1.WaitGroup核心源码分析
```
type WaitGroup struct {
    noCopy noCopy
    state1 [3]uint32 // 包含counter, waiter计数和信号量
}

func (wg *WaitGroup) Add(delta int) {
    state := atomic.AddUint64(statep, uint64(delta)<<32)
    v := int32(state >> 32) // counter
    w := uint32(state)      // waiter count
    
    if v < 0 {
        panic("sync: negative WaitGroup counter")
    }
    if w != 0 && delta > 0 && v == int32(delta) {
        panic("sync: WaitGroup misuse: Add called concurrently with Wait")
    }
    if v > 0 || w == 0 {
        return
    }
    // 计数器为0且有等待者，唤醒所有等待者
    *statep = 0
    for ; w != 0; w-- {
        runtime_Semrelease(semap, false, 0)
    }
}

func (wg *WaitGroup) Wait() {
    for {
        state := atomic.LoadUint64(statep)
        v := int32(state >> 32) // counter
        if v == 0 {
            return
        }
        // 增加等待者计数
        if atomic.CompareAndSwapUint64(statep, state, state+1) {
            runtime_Semacquire(semap)
            if *statep != 0 {
                panic("sync: WaitGroup is reused before previous Wait has returned")
            }
            return
        }
    }
}

func (wg *WaitGroup) Done() {
    wg.Add(-1)
}

```

对于waitGroup主要的核心方法就3个，分别为Add(),Wait(),Done()
设计上:
原子操作：所有状态变更都通过atomic包实现，保证并发安全

状态合并：将counter和waiter计数合并到一个64位整数中：

高32位：counter（Add/Done操作）

低32位：waiter计数（Wait操作）

源码时序图:
调用Add()时:
   |--原子增加counter值
   |--检查counter有效性
   |--如果有等待者且counter归零->唤醒所有等待者

调用Wait()时:
   |--检查counter是否为0
   |--如果不为0:
       |--原子增加waiter计数
       |--阻塞等待信号量
       |--被唤醒后检查状态

调用Done()时:
   |--相当于Add(-1)
   |--当counter归零时:
       |--重置状态
       |--释放信号量唤醒所有等待者



## 2.Mutex源码分析
数据结构:
```
type Mutex struct {
    state int32  // 锁状态复合字段
    sema  uint32 // 信号量，用于阻塞/唤醒goroutine
}
```
state状态位如下:
| 31.....................3 |    2    |    1    |    0    |
|--------------------------|---------|---------|---------|
|  等待goroutine数量        | 唤醒标记 | 饥饿模式 | 锁定状态 |

关键常量定义如下:
```
const (
    mutexLocked = 1 << iota // 1（锁定状态）
    mutexWoken             // 2（唤醒标记）
    mutexStarving          // 4（饥饿模式）
    mutexWaiterShift = iota // 3（等待者数量位移）
    
    starvationThresholdNs = 1e6 // 1ms，进入饥饿模式的等待时间阈值
)

```

通过对于state进行各种移位操作，使得代码变得非常高内聚，实现上很优雅

```
func (m *Mutex) Lock() {
    // 快速路径：直接CAS获取锁（无竞争情况）
    if atomic.CompareAndSwapInt32(&m.state, 0, mutexLocked) {
        return
    }
    
    // 慢路径（有竞争情况）
    m.lockSlow()
}
```
1.快速路径，若没有锁的情况，直接通过CAS获取
2.有竞争的情况，进行饥饿模式和公平模式的抢占锁逻辑

### 无竞争加锁
goroutine1           Mutex.state
   |                    0
   |---Lock()-----------|
   | CAS(0,1)---------->1
   |<---成功返回---------|

### 有竞争加锁
goroutine1           goroutine2           Mutex.state
   |                     |                    1
   |---Lock()------------|                    |
   |                     |---Lock()-----------|
   |                     |---进入slow path----|
   |                     |---加入等待队列---->1+8=9(1等待者)
   |---Unlock()----------|
   | state=9-1=8-------->|
   |---unlockSlow()------|
   |---唤醒goroutine2----|
   |                     |<---获取锁---------1

### 饥饿模式
goroutine1 (持有锁长时间不释放)
   |
goroutine2 等待超过1ms --> 触发饥饿模式
   |
goroutine3 新来 --> 直接加入队列尾部
   |
goroutine1 释放锁 --> 直接交给goroutine2


### 性能优化点
快速路径：无竞争时只需一次CAS操作

自旋等待：减少上下文切换开销

公平性控制：通过饥饿模式防止长等待

位压缩：将多个状态压缩到一个32位整数

信号量优化：精确控制唤醒机制


## 3.Once源码分析
```
type Once struct {
    done uint32 // 执行状态标志
    m    Mutex // 互斥锁
}
```

* done：原子标志位，0表示未执行，1表示已执行
* 互斥锁，用于保护慢路径的并发控制

核心代码:
```
func (o *Once) Do(f func()) {
    // 快速路径检查：原子加载done标志
    if atomic.LoadUint32(&o.done) == 0 {
        // 慢路径（实际执行逻辑）
        o.doSlow(f)
    }
}
```

慢路径的实现:
```
func (o *Once) doSlow(f func()) {
    o.m.Lock()
    defer o.m.Unlock()
    
    if o.done == 0 {
        defer atomic.StoreUint32(&o.done, 1)
        f()
    }
}
```

双检查锁定模式：

外层检查（原子读）：快速过滤已执行情况

内层检查（加锁后）：确保真正未执行

defer 的巧妙使用：

defer atomic.StoreUint32保证即使f() panic也能标记为已执行

defer o.m.Unlock()确保锁一定会释放


## 4.SingleFlight源码分析
Group结构体
```
type Group struct {
    mu sync.Mutex       // 保护 m 的互斥锁
    m  map[string]*call // 正在处理的请求映射表
}
```

call结构图
```
type call struct {
    wg  sync.WaitGroup // 用于同步等待
    val interface{}    // 函数返回值
    err error          // 函数返回错误
    forgotten bool     // 是否已遗忘
    dups  int         // 重复调用计数
    chans []chan<- Result // 结果广播通道
}
```

Do方法
```
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
    g.mu.Lock()
    if g.m == nil {
        g.m = make(map[string]*call)
    }
    
    // 检查是否已有相同 key 的请求在处理
    if c, ok := g.m[key]; ok {
        c.dups++
        g.mu.Unlock()
        c.wg.Wait() // 等待已有请求完成
        return c.val, c.err
    }
    
    // 创建新的 call
    c := new(call)
    c.wg.Add(1)
    g.m[key] = c
    g.mu.Unlock()

    // 执行实际函数
    c.val, c.err = fn()
    c.wg.Done()

    // 清理
    g.mu.Lock()
    delete(g.m, key)
    g.mu.Unlock()

    return c.val, c.err
}
```
本质上是通过map进行判断，是否有同样的key进行请求;
互斥锁 (mu)：保护对 map 的并发访问

WaitGroup (wg)：协调重复请求的等待

原子性操作：map 操作和结果赋值都受锁保护


## 5.Pool源码分析
关键设计:
1.无竞争场景
优先从当前P的private无锁获取

其次尝试从当前P的shared无锁弹出

最后回退到victim缓存或New创建

2.有竞争场景
从其他P的shared队列尾部偷取（LIFO）

使用CAS操作保证并发安全

偷取失败后回退到victim或New

