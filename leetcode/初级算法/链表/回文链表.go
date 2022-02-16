package main

/*
给你一个单链表的头节点 head ，请你判断该链表是否为回文链表。如果是，返回 true ；否则，返回 false 。

 

示例 1：


输入：head = [1,2,2,1]
输出：true
示例 2：


输入：head = [1,2]
输出：false
 

提示：

链表中节点数目在范围[1, 105] 内
0 <= Node.val <= 9
 

进阶：你能否用 O(n) 时间复杂度和 O(1) 空间复杂度解决此题？

 */

func main() {

}


/**
 * Definition for singly-linked list.
 * type ListNode struct {
 *     Val int
 *     Next *ListNode
 * }
 */


// O(1) 解决
func isPalindrome(head *ListNode) bool {


	if head == nil {
		return true
	}
	//快慢指针 找到中间
	qianmiande := findMid(head)
	// 反转链表
	houmiande := fanzhuan(qianmiande.Next)

	p1 := head
	p2 := houmiande

	for p2 != nil {
		if p1.Val == p2.Val {
			p1 = p1.Next
			p2 = p2.Next
			continue
		}
		return false
	}
	return true
}


func findMid(head *ListNode) *ListNode {
	fast := head
	slow := head

	for fast.Next != nil && fast.Next.Next != nil {
		fast = fast.Next.Next
		slow = slow.Next
	}
	return slow
}

func fanzhuan(head *ListNode) *ListNode {
	var cur, tou *ListNode = head, nil

	for cur != nil {
		var next = cur.Next
		cur.Next = tou
		tou = cur
		cur = next
	}

	return tou
}
