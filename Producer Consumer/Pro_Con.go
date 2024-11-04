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

func producer(ch chan int) {
    for i := 0; i < 10; i++ {
        time.Sleep(time.Second)
        fmt.Println("Producer: sending", i)
        ch <- i
    }
    close(ch)
}

func consumer(ch <-chan int) {
    for i := range ch {
        time.Sleep(time.Second)
        fmt.Println("Consumer: receiving", i)
    }
}

func main() {
    ch := make(chan int)
    go producer(ch)
    go consumer(ch)
    time.Sleep(time.Second * 10)
}