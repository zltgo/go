package main

import (
	"fmt"
	"github.com/pkg/errors"
)

func f(i int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			err = errors.New("222")
		}
	}()

	panic(i)
	return errors.New("111")
}

 type test1 struct{
	I int
}

func(t *test1) reset() {
	t = &test1{3}
}
func main() {
	lru.
}
