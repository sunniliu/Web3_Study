# Channel学习

## 1.基本概念
* channel用于不同的goroutine之间安全传递数据，类似"数据管道"的功能
* 线程安全的，在channel的数据结构中已经有了锁，不需要额外加锁
* channel是FIFO的结构

## 2.channel的基本操作
### 2.1 声明和创建
    // 声明一个传递 int 类型的 channel（未初始化，值为 nil）
    var ch chan int

    // 使用 make 初始化一个无缓冲 channel
    ch := make(chan int)

    // 初始化一个缓冲大小为 10 的 channel
    bufferedCh := make(chan int, 10)