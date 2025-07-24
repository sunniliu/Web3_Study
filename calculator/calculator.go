package main

import (
	"flag"
	"fmt"
)

func main() {
	num1 := flag.Float64("num1", 0, "num1")
	num2 := flag.Float64("num2", 0, "num2")
	op := flag.String("op", "+", "运算符:+ - * /")
	flag.Parse()

	// 计算结果
	var result float64

	switch *op {
	case "+":
		result = *num1 + *num2
	case "-":
		result = *num1 - *num2
	case "*":
		result = (*num1) * (*num2)
	case "/":
		if *num2 == 0 {
			fmt.Println("error: num2 is 0")
		}
		result = *num1 / *num2
	default:
		fmt.Println("panic")
	}
	fmt.Printf("结果: %.2f %s %.2f = %.2f\n", *num1, *op, *num2, result)
}
