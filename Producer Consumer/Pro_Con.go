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

// producer sends integers to the channel 'ch'.
// It runs in a loop from 0 to 9, simulating a delay with time.Sleep.
// After sending all values, it closes the channel.
func producer(ch chan int) {
    for i := 0; i < 10; i++ {
        time.Sleep(time.Second) // Simulate some work with a 1-second delay
        fmt.Println("Producer: sending", i) // Log the value being sent
        ch <- i // Send the value to the channel
    }
    close(ch) // Close the channel to signal no more values will be sent
}

// consumer receives integers from the read-only channel 'ch'.
// It processes values received from the channel until it is closed.
func consumer(ch <-chan int) {
    for i := range ch { // Read values from the channel until it's closed
        time.Sleep(time.Second) // Simulate some work with a 1-second delay
        fmt.Println("Consumer: receiving", i) // Log the value being received
    }
}

func main() {
    ch := make(chan int) // Create an unbuffered channel of type int
    go producer(ch) // Start the producer goroutine
    go consumer(ch) // Start the consumer goroutine
    time.Sleep(time.Second * 10) // Allow some time for producer and consumer to complete their work
}