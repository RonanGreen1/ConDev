//--------------------------------------------
// Author: Ronan Green (Joseph.Kehoe@setu.ie)
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
	"sync"
	"time"
)



func Producer(){

}

func Consumer(){

}

func Shelf(){

}

func main(){
	totalRoutines := 10
	var wg sync.WaitGroup
	wg.Add(totalRoutines)
	var theLock sync.Mutex
	For n := range totalRoutines{
		go Shelf(&wg, theLock, n )
	}

}