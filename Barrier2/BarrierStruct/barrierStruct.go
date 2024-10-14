package main

import (
     "fmt"
     "sync"
     "time"
     "math/rand/v2"
)


//Create a barrier data type
type barrier struct{
     theChan chan bool
     theLock sync.Mutex
     total int
     count int
}

//creates a properly initialised barrier
//N== number of threads (go Routines)
func createBarrier(N int) barrier{
     theBarrier:= barrier{
         theChan : make(chan bool),
         total: N,
         count : 0,
     }
     return theBarrier
}

//Method belonging to barrier data type
//Blocks until everyone reaches this point then lets everyone continue
func (b *barrier) wait(){
     b.theLock.Lock()
     b.count++
     if b.count== b.total{
         b.theLock.Unlock()
         fmt.Println("here")
         for _= range b.total-1{
             <- b.theChan
         }
     }else{
         fmt.Println(b.count)
         b.theLock.Unlock()
         b.theChan <- true
     }
}//wait

func WorkWithRendezvous(wg *sync.WaitGroup, Num int, theBarrier *barrier) bool {
     var X time.Duration
     X=time.Duration(rand.IntN(5))
     time.Sleep(X * time.Second)//wait random time amount
     fmt.Println("Part A", Num)
     //Rendezvous here
     theBarrier.wait()
     fmt.Println("PartB",Num)
     wg.Done()
     return true
}



func main() {
     var wg sync.WaitGroup
     barrier := createBarrier(5)
     threadCount:=5

     wg.Add(threadCount)
     for N := range threadCount {
         go WorkWithRendezvous(&wg, N,&barrier)
     }
     wg.Wait() //wait here until everyone (5 go routines) is done

}
