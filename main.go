
package main

import (
	"fmt"
)

func twoSum(nums []int, target int) []int {
	res := []int{}
	for i, val1 := range nums {
		for j, val2 := range nums {
			if len(res) > 1 {
				break
			} else {
				if i != j {
					if val1+val2 == target {
						res = append(res, i)
						res = append(res, j)
					}
				}
			}
		}
	}
	return res
}
func main() {
	nums := []int{6, -4, 3, 1, 4, 2}
	target := 2
	fmt.Println(twoSum(nums, target))
}
