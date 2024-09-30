// Ronan Green
// C00270395

package main

import (
	"errors"
	"fmt"
	"regexp"
)

func main() {

	// This program is used for transfering roman numerials into base 10 decimal numbers.
	// To run this code make sure to have go downloaded then just clone the repo and open the folder in the ternimal.
	// Then run go run main.go when you have cd into the folder.

	var romanNumeral string //string of roman numerals input by user

	convertion := map[string]int{
		"I":  1,
		"V":  5,
		"X":  10,
		"L":  50,
		"C":  100,
		"D":  500,
		"M":  1000,
		"IV": 4,
		"IX": 9,
		"XL": 40,
		"XC": 90,
		"CD": 400,
		"CM": 900,
	}

	fmt.Println("Enter Roman Numerials")
	fmt.Scanln(&romanNumeral)

	if len(romanNumeral) > 15 {
		fmt.Println("Error: Input cannot be more than 15 characters.")
		return
	}

	// Define allowed characters (Roman numerals: I, V, X, L, C, D, M),
	// Return only boolean 'matched' then check if parameters were met
	validInputPattern := "^[IVXLCDM]+$"
	matched, _ := regexp.MatchString(validInputPattern, romanNumeral)

	if !matched {
		fmt.Println("Incorrect Input, please follow the following parameters:\nOnly use the following Roman numeral characters: I, V, X, L, C, D, M")
		return
	}

	result, err := calculation(romanNumeral, convertion)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println(romanNumeral, "=", result)
	}
}

func calculation(romanNumeral string, convertion map[string]int) (int, error) {
	var total int //number of roman numerals entered
	length := len(romanNumeral)
	var count int = 1

	//initial check of 2 characters to see if they match any entries in the map
	for i := 0; i < length; i++ {
		if i+1 < length {
			chars := romanNumeral[i : i+2]
			// value is assigned the same as chars and exists is a boolean for whether value exists in the map
			// This is used to check for instances where the first didget subtracts from the second
			if value, exists := convertion[chars]; exists {
				total += value
				i++
				continue
			}

		}

		char := string(romanNumeral[i])
		// Add single characters with a check to make sure the character isn't higher than the next.
		//This also checks for combinations like VV or DD or a letter shwoing up more than 4 times like IIII
		if value, exists := convertion[char]; exists {
			if i+1 < length && convertion[char] < convertion[string(romanNumeral[i+1])] {
				return -1, errors.New("invalid Roman numeral combination")
			}else if i+1 < length && char == "V" && string(romanNumeral[i + 1]) == "V"{
				return -1, errors.New("invalid Roman numeral combination")
			}else if i+1 < length && char == "L" && string(romanNumeral[i + 1]) == "L"{
				return -1, errors.New("invalid Roman numeral combination")
			}else if i+1 < length && char == "D" && string(romanNumeral[i+ 1]) == "D"{
				return -1, errors.New("invalid Roman numeral combination")
			}else if i+1 < length && char == string(romanNumeral[i + 1]){
				count += 1
				if count > 3{
					return -1, errors.New("invalid Roman numeral combination")
				}
			}else if i+1 < length && char != string(romanNumeral[i + 1]){
				count = 1
			}
			
			total += value
			
		}
	}
	return total, nil
}
