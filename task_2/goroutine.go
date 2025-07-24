package main

import (
	"fmt"
	"sync"
	"time"
)

func exexample_1() {
	go func() {
		for i := 0; i < 10; i++ {
			if i%2 == 0 {
				fmt.Printf("value is %d\n", i)
			}
		}
	}()

	go func() {
		for i := 0; i < 10; i++ {
			if i%2 == 1 {
				fmt.Printf("value is %d\n", i)
			}
		}
	}()
	time.Sleep(2 * time.Second)
}

func exexample_2() {
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(time.Duration(id+1) * time.Second)
			fmt.Printf("task is finished %d\n", id+1)
		}(i)
	}
	wg.Wait()
	time.Sleep(2 * time.Second)
}

func main() {
}
