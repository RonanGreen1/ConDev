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
	"math/rand"
	"sync"
	"time"
)

// Philosopher represents a philosopher with an ID and two forks (left and right).
type Philosopher struct {
	Id        int
	LeftFork  *sync.Mutex
	RightFork *sync.Mutex
}

const (
	NOfPhilosophers = 5 // Number of philosophers at the table
)

func main() {
	var wg sync.WaitGroup
	wg.Add(NOfPhilosophers)
	// Create an array of forks (mutexes) for each philosopher.
	var forks [NOfPhilosophers]*sync.Mutex
	for i := 0; i < NOfPhilosophers; i++ {
		forks[i] = &sync.Mutex{} // Initialize each fork as a mutex
	}

	// Create a slice of philosophers and assign forks to each philosopher.
	philosophers := make([]*Philosopher, NOfPhilosophers)
	for i := 0; i < NOfPhilosophers; i++ {
		// Each philosopher gets a left fork and a right fork (next fork in the circle).
		philosophers[i] = &Philosopher{
			Id:        i + 1, // Philosopher IDs are 1-based
			LeftFork:  forks[i],
			RightFork: forks[(i+1)%NOfPhilosophers], // Right fork is the next one in the circle
		}
	}

	// Start a goroutine for each philosopher to dine concurrently.
	for _, phil := range philosophers {
		go func(p *Philosopher) {
			defer wg.Done() // Mark this goroutine as done when finished
			for {           // Each philosopher eats 3 times
				p.dine() // Philosopher goes through the dine process
			}
		}(phil)
	}

	// Wait for all philosophers to finish dining.
	wg.Wait()
	fmt.Println("All philosophers have finished dining.")
}

// dine represents the philosopher's process of thinking, acquiring forks, eating, and releasing forks.
func (p *Philosopher) dine() {
	p.think() // Philosopher thinks before attempting to eat

	// Lock the left fork first, then the right fork to start eating.
	p.LeftFork.Lock()
	p.RightFork.Lock()

	p.eat() // Philosopher eats after acquiring both forks

	// Unlock the right fork first, then the left fork after eating.
	p.RightFork.Unlock()
	p.LeftFork.Unlock()
}

// think simulates the philosopher thinking for a random amount of time.
func (p *Philosopher) think() {
	t := time.Duration(rand.Intn(3e3)) * time.Millisecond // Random thinking time between 0 and 1 second
	fmt.Printf("Philosopher %d is thinking for %v\n", p.Id, t)
	time.Sleep(t) // Simulate thinking by sleeping
}

// eat simulates the philosopher eating for a random amount of time.
func (p *Philosopher) eat() {
	t := time.Duration(rand.Intn(3e3)) * time.Millisecond // Random eating time between 0 and 1 second
	fmt.Printf("Philosopher %d is eating for %v\n", p.Id, t)
	time.Sleep(t) // Simulate eating by sleeping
}
