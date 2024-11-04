//--------------------------------------------
// Author: Ronan Green
// Created on 14/10/2024
// Modified by: Ronan Green
// Description:
// 
// Issues:
// 
//
//--------------------------------------------

package main

import (
    "fmt"
    "time"
)

// producer sends a series of integers to the provided channel.
// The channel is closed after all values are sent.
func producer(ch chan int) {
    for i := 0; i < 10; i++ {
        time.Sleep(time.Second) // Simulate some work by sleeping for 1 second.
        fmt.Println("Producer: sending", i)
        ch <- i // Send the current value of i to the channel.
    }
    close(ch) // Close the channel to indicate that no more values will be sent.
}

// consumer reads values from the provided channel until it is closed.
func consumer(ch <-chan int) {
    for i := range ch { // Iterate over values received from the channel.
        time.Sleep(time.Second) // Simulate some work by sleeping for 1 second.
        fmt.Println("Consumer: receiving", i)
    }
}

func main() {
    ch := make(chan int) // Create an unbuffered channel of type int.
    go producer(ch) // Start the producer goroutine.
    go consumer(ch) // Start the consumer goroutine.
    
    // Sleep to allow enough time for producer and consumer to complete.
    // Note: This approach may not guarantee proper synchronization.
    time.Sleep(time.Second * 12) // Increased sleep time to ensure both producer and consumer complete.
    
    // To ensure all goroutines finish properly, consider using sync.WaitGroup or other synchronization methods.
}