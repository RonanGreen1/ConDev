package main

import (
	"fmt"
	"regexp"
)

func main() {

	// This program i used for transfering roman numerials into base 10 decimal numbers

	var roman_numerial string

	fmt.Println("Enter Roman Numerials")
	fmt.Scanln(&roman_numerial)

	if len(roman_numerial) > 15 {
		fmt.Println("Error: Input cannot be more than 15 characters.")
		return
	}

	// Define allowed characters (Roman numerals: I, V, X, L, C, D, M),
	// Return only boolean 'matched' then check if parameters were met
	validInputPattern := "^[IVXLCDMivxlcdm]+$"
	matched, _ := regexp.MatchString(validInputPattern, roman_numerial)

	if !matched {
		fmt.Println("Incorrect Input, please follow the following parameters:\nOnly use the following Roman numeral characters: I, V, X, L, C, D, M")
	} else {
		
	}

}
