package main

import(
	"fmt"
)

func main(){

	array :=[...]int{1,2,3}
	fmt.Println(array)

	slice :=[]int{1,2,3}
	fmt.Println(cap(slice))

	slice2 := array[1:2]
	fmt.Println(slice2)

	slice3 := make([]int, 10)
	fmt.Println(slice3)



}