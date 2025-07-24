package main

import (
	"fmt"
	"math"
)

type Shape interface {
	Area() float64
	Perimeter() float64
}

type Rectangle struct {
	width, height float64
}

type Circle struct {
	radius float64
}

func (r Rectangle) Area() float64 {
	return r.width * r.height
}

func (r Rectangle) Perimeter() float64 {
	return 2 * (r.width + r.height)
}

func (c Circle) Area() float64 {
	return math.Pi * c.radius * c.radius
}

func (c Circle) Perimeter() float64 {
	return 2 * math.Pi * c.radius
}

type Person struct {
	name string
	age  int
}

type Employee struct {
	person     Person
	employeeId int
}

func (e Employee) Print() {
	fmt.Println(e.person.name)
	fmt.Println(e.person.age)
	fmt.Println(e.employeeId)
}

func main() {
	c := Circle{radius: 10}
	fmt.Println(c.Area())
	fmt.Println(c.Perimeter())

	r := Rectangle{width: 10, height: 5}
	fmt.Println(r.Area())
	fmt.Println(r.Perimeter())

	e := Employee{Person{"Jack", 20}, 30}
	e.Print()

}
