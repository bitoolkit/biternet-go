package client_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bitoolkit/biternet-go/client"
)

func ExampleClient_Status() {
	c, err := client.New(client.Config{})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	state, err := c.Status(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(state.Online(), state.IP)
}
