package main

import "fmt"

/*

给你一个数组，将数组中的元素向右轮转 k 个位置，其中 k 是非负数。

 

示例 1:

输入: nums = [1,2,3,4,5,6,7], k = 3
输出: [5,6,7,1,2,3,4]
解释:
向右轮转 1 步: [7,1,2,3,4,5,6]
向右轮转 2 步: [6,7,1,2,3,4,5]
向右轮转 3 步: [5,6,7,1,2,3,4]
示例 2:

输入：nums = [-1,-100,3,99], k = 2
输出：[3,99,-1,-100]
解释:
向右轮转 1 步: [99,-1,-100,3]
向右轮转 2 步: [3,99,-1,-100]
 

提示：

1 <= nums.length <= 105
-231 <= nums[i] <= 231 - 1
0 <= k <= 105

 */

func main() {
	nums := []int{1,2,3,4,5,6,7}
	k := 3
	k %= len(nums)
	fmt.Println(len(nums))
	res(nums)
	res(nums[:k])
	res(nums[k:])
	fmt.Println(nums)
}

func res(n []int)  {
	for i, l := 0, len(n); i < l/2 ; i++ {
		n[i], n[l-i-1] = n[l-i-1], n[i]
	}
}