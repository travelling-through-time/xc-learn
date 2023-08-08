package main

import "fmt"

func main() {
	merge([]int{1, 2, 3, 0, 0, 0}, 3, []int{2, 5, 6}, 3)
}

func merge(nums1 []int, m int, nums2 []int, n int) {
	//Go语言的内置函数 copy() 可以将一个数组切片复制到另一个数组切片中，如果加入的两个数组切片不一样大，就会按照其中较小的那个数组切片的元素个数进行复制。
	//slice1 := []int{1, 2, 3, 4, 5}
	//slice2 := []int{5, 4, 3}
	//copy(slice2, slice1) // 只会复制slice1的前3个元素到slice2中
	//copy(slice1, slice2) // 只会复制slice2的3个元素到slice1的前3个位置
	//copy(nums1[m:], nums2)
	//fmt.Print(nums1)
	sorted := make([]int, 0, m+n)
	p1, p2 := 0, 0
	for {
		if nums1[p1] > nums2[p2] {
			sorted = append(sorted, nums2[p2])
			p2++
		} else {
			sorted = append(sorted, nums1[p1])
			p1++
		}
		if p1 == m {
			sorted = append(sorted, nums2[p2:]...)
			break
		}
		if p2 == n {
			sorted = append(sorted, nums1[p1:]...)
			break
		}
	}
	fmt.Print(sorted)
	copy(nums1, sorted)
	fmt.Print(nums1)
}
