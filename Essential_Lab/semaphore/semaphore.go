package main

import (
    "fmt"
    "sync"
    "time"
)
//make struct containing channel
//add init, acquire and release
//type semaphore struct {
//	theCounter chan struct{}
//}


func main() {
    maxGoroutines := 5
    semaphore := make(chan struct{}, maxGoroutines)

    var wg sync.WaitGroup
    for i := 0; i < 20; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            // Simulate a task
            fmt.Printf("Running task %d\n", i)
            time.Sleep(2 * time.Second)
        }(i)
    }
    wg.Wait()
}
