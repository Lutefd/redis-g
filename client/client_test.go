package client

import (
	"context"
	"fmt"
	"log"
	"testing"
)

func TestNewClient(t *testing.T) {
	c, err := New("localhost:5001")
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		fmt.Println("SET =>", fmt.Sprintf("bar %d", i))
		if err := c.Set(context.TODO(), fmt.Sprintf("foo %d", i), fmt.Sprintf("bar %d", i)); err != nil {
			log.Fatal(err)
		}
		val, err := c.Get(context.TODO(), fmt.Sprintf("foo %d", i))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("GET =>", val)
	}
}
