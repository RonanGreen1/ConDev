//Lab 3 Barrier © 2024 by Ronan Green is licensed under CC BY-NC 4.0. To view a copy of this license, visit https://creativecommons.org/licenses/by-nc/4.0/

//--------------------------------------------
// Author: Joseph Kehoe (Joseph.Kehoe@setu.ie)
// Created on 30/9/2024
// Modified by: Ronan Green
// Issues:
// The barrier is not implemented!
//--------------------------------------------

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// Place a barrier in this function --use Mutex's and Semaphores
func doStuff(goNum int, wg *sync.WaitGroup, arrived *int, max int, thelock *sync.Mutex, sem *semaphore.Weighted, ctx context.Context) bool {
	time.Sleep(time.Second)
	fmt.Println("Part A", goNum)
	thelock.Lock()
	*arrived++
	if *arrived == max {
		//we wait here until everyone has completed part A
		sem.Release(1)
		thelock.Unlock()
		sem.Acquire(ctx, 1)
	} else {
		thelock.Unlock()
		sem.Acquire(ctx, 1)
		sem.Release(1)
	}
	fmt.Println("PartB", goNum)
	wg.Done()
	return true
}

func main() {
	totalRoutines := 10
	arrived := 0
	var wg sync.WaitGroup
	wg.Add(totalRoutines)
	//we will need some of these
	ctx := context.TODO()
	var theLock sync.Mutex
	sem := semaphore.NewWeighted(0)
	//sem.Acquire(ctx, 1)
	for i := range totalRoutines { //create the go Routines here
		go doStuff(i, &wg, &arrived, totalRoutines, &theLock, sem, ctx)
	}
	//sem.Release(1)

	wg.Wait() //wait for everyone to finish before exiting
}
