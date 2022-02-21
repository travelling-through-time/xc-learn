package main

/*
给你一个二叉树的根节点 root ， 检查它是否轴对称。

 

示例 1：


输入：root = [1,2,2,3,4,4,3]
输出：true
示例 2：


输入：root = [1,2,2,null,3,null,3]
输出：false
 

提示：

树中节点数目在范围 [1, 1000] 内
-100 <= Node.val <= 100

 */

func main() {
	
}

/**
 * Definition for a binary tree node.
 * type TreeNode struct {
 *     Val int
 *     Left *TreeNode
 *     Right *TreeNode
 * }
 */

func isSymmetric(root *TreeNode) bool {
	if root == nil {
		return true
	}
	// 从两个字节点开始判断
	return isSymmetricHelper(root.Left, root.Right);
}

func isSymmetricHelper(left *TreeNode, right *TreeNode) bool {
	// 如果左右子节点都为空，说明当前节点是叶子节点，返回true
	if left == nil && right == nil {
		return true
	}
	// 如果当前及诶单只有一个子节点 或者 两个子节点，但两个子节点的值不相同，直接返回false
	if left == nil || right == nil || left.Val != right.Val {
		return false
	}
	// 然后左子节点的左子节点和右子节点的右子节点比较，左子节点的右子节点和右子节点的左子节点比较
	return isSymmetricHelper(left.Left, right.Right) && isSymmetricHelper(left.Right, right.Left)
}