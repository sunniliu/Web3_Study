package main

import (
	"flag"
	"fmt"
	"sync"
	"time"
)

func exeample_1() {
	ch := make(chan int)
	go func() {
		for i := 0; i < 10; i++ {
			ch <- i
		}
		close(ch)
	}()

	go func() {
		for num := range ch {
			fmt.Printf("%d ", num)
		}
	}()
	time.Sleep(1 * time.Second)
}

func exeample_2() {
	ch := make(chan int, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 1; i < 100; i++ {
			ch <- i
			fmt.Printf("value is %d\n", i)
		}
		close(ch)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for num := range ch {
			fmt.Printf("%d ", num)
		}
	}()

	wg.Wait()
	fmt.Println("done")
}

func main() {
	name := flag.String("name", "Guest", "Your name")
	flag.Parse()
	fmt.Println("Hello", *name)
}
