package main

import (
	"fmt"
	"strings"
	"unicode"
)

/*
给定一个字符串，验证它是否是回文串，只考虑字母和数字字符，可以忽略字母的大小写。

说明：本题中，我们将空字符串定义为有效的回文串。

 

示例 1:

输入: "A man, a plan, a canal: Panama"
输出: true
解释："amanaplanacanalpanama" 是回文串
示例 2:

输入: "race a car"
输出: false
解释："raceacar" 不是回文串
 

提示：

1 <= s.length <= 2 * 105
字符串 s 由 ASCII 字符组成

 */

func main() {
	s := "A man, a plan, a canal: Panama"
	for _,i := range s {
		fmt.Println(unicode.IsLetter(i))
	}
}

func isPalindrome(s string) bool {
	p := 0
	q := len(s) - 1
	for {
		if p >= q {
			return true
		}
		if unicode.IsLetter(int32(s[p])) && unicode.IsLetter(int32(s[q])){
			if strings.ToLower(string(s[p])) == strings.ToLower(string(s[q])) {
				p++
				q++
			} else {
				return false
			}
		}
	}
}

func isPalindrome1(s string) bool {
	s = strings.ToLower(s)
	l, r := 0, len(s)-1

	for l <= r {
		for l <= r && !isAlnum(s[l]) {
			l++
		}
		for l <= r && !isAlnum(s[r]) {
			r--
		}
		if l <= r && s[l] != s[r] {
			return false
		}
		l++
		r--
	}

	return true
}

func isAlnum(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z')
}

func isPalindrome2(s string) bool {
	l,r:=0,len(s)-1
	for l<r{
		for  l<len(s) && notIs(s[l]){
			l++
		}
		for r>0 && notIs(s[r]){
			r--
		}

		if l<len(s) && r>0 && toUpper(s[l])!=toUpper(s[r]){
			return false
		}
		//只要不是空格和逗号直接加1
		l++
		r--
	}
	return true
}

func toUpper(str byte)byte{
	//大写 65-90
	//小写 97-122
	if str>=97 && str<=122{
		return str-32
	}
	return str
}

func notIs(str byte)bool{
	if (str>='a' && str<='z') || (str>='A' && str<='Z') || (str>='0' && str<='9'){
		return false
	}
	return true
}