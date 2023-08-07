package main

import (
	"fmt"
	"math/rand"
	"time"
)


func main() {

	v := RandomSequence(2,12)
	fmt.Printf("生成的随机序列 size:%d,nmber[]:%v\n", len(v),v)
	// 堆排序
	v = []int{3,2,1,5,6,4}
	heapsort(v)

	fmt.Printf("排序后 size:%d,nmber[]:%v\n", len(v),v)
}



func RandomSequence(min,max int) []int{
	//计算序列的长度
	lenghth := max - min + 1

	//初始化一个长度为lenghth的原始切片，初始值从min到max
	initArr := make([]int,lenghth)
	for i :=0; i < lenghth; i++{
		initArr[i] = i + min
	}

	//初始化一个长度为lenghth的目标切片
	rtnArr := make([]int,lenghth)

	//初始化随机种子
	rand.Seed(time.Now().Unix())

	//生成目标序列
	for i :=0; i < lenghth; i++{
		//生成一个随机序号
		index := rand.Intn(lenghth - i)

		//将原始切片中序号index对应的值赋给目标切片
		rtnArr[i] = initArr[index]

		//替换掉原始切片中使用过的下标index对应的值
		initArr[index] = initArr[lenghth - i - 1]
	}

	return rtnArr
}

func heapsort(arr []int) []int {
	arrLen := len(arr)
	buildMaxHeap(arr, arrLen)
	// 循环数组长度
	for i := arrLen - 1; i >= 0 ; i-- {
		// 最大放后面
		swap(arr, 0, i)
		arrLen -= 1
		// 调整大顶堆 堆化（heapify）
		heapify(arr, 0, arrLen)

	}

	return arr
}

func heapify(arr []int, i, arrLen int)  {
	// 非叶子节点的左节点 索引
	left := 2*i + 1
	// 右节点 索引
	right := left + 1
	// 最大值索引
	largest := i

	if left < arrLen && arr[left] > arr[largest] {
		largest = left
	}

	if right < arrLen && arr[right] > arr[largest] {
		largest = right
	}

	// 	最大的和 i 索引不等 需要在调整堆
	if largest != i  {
		swap(arr, i, largest)
		heapify(arr, largest, arrLen)
	}

}


func buildMaxHeap(arr []int, arrLen int)  {
	for i := arrLen / 2; i >= 0; i-- {
		heapify(arr, i, arrLen)
	}
}


func swap(arr []int, i, j int) {
	arr[i], arr[j] = arr[j], arr[i]
}

