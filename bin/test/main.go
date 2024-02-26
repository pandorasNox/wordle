package main

import "fmt"

type point struct {
	x int
	y int
}

func (p point) String() string {
	return fmt.Sprintf("[x:%d,y:%d]", p.x, p.y)
}

type points []point

func (ps points) String() string {
	var r string
	for _, p := range ps {
		r += fmt.Sprintf("%s", p)
	}
	return r
}

func main() {
	ps := points{
		{},
		{1, 1},
	}

	fmt.Printf("points:\n%s\n", ps)

	change(&ps)

	fmt.Printf("points:\n%s\n", ps)
}

func change(ps *points) {
	val := &(*ps)[1]
	val.x = 5
}
