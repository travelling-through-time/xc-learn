package main

/*
给你二叉树的根节点 root ，返回其节点值的 层序遍历 。 （即逐层地，从左到右访问所有节点）。

 

示例 1：


输入：root = [3,9,20,null,null,15,7]
输出：[[3],[9,20],[15,7]]
示例 2：

输入：root = [1]
输出：[[1]]
示例 3：

输入：root = []
输出：[]
 

提示：

树中节点数目在范围 [0, 2000] 内
-1000 <= Node.val <= 1000

 */

/**
 * Definition for a binary tree node.
 * type TreeNode struct {
 *     Val int
 *     Left *TreeNode
 *     Right *TreeNode
 * }
 */

func main() {
	//root := TreeNode{}[3,9,20,null,null,15,7]
}
func levelOrder(root *TreeNode) [][]int {
	ret := [][]int{}

	if root == nil {
		return ret
	}

	q := []*TreeNode{root}
	for i := 0; len(q) > 0 ; i++ {
		ret = append(ret, []int{})

		p := []*TreeNode{}

		for j := 0; j < len(q); j++ {
			node := q[j]
			ret[i] = append(ret[i], node.Val)
			// 对原有数组进行扩容
			/*
			[[]]
			[[3] []]
			[[3] [9 20] []]
			 */

			if node.Left != nil {
				p = append(p, node.Left)
			}

			if node.Right != nil {
				p = append(p, node.Right)
			}
		}
		q = p
	}
	return ret
}

