package main

import "fmt"

func main() {
	fmt.Print(removeElement([]int{3, 2, 2, 3}, 3))
}

func removeElement(nums []int, val int) int {
	j := 0
	
	for i, num := range nums {
		if num == val {
			nums[i], nums[i+1] = nums[i+1], nums[i]
			j++
		}
	}
	return len(nums) - j
}
