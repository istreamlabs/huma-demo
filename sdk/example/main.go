package main

import (
	"context"
	"fmt"
	"time"

	"example.com/demo/sdk"
)

func main() {
	client, _ := sdk.NewClientWithResponses("http://localhost:8888")

	resp, err := client.ListChannelsWithResponse(context.Background(), nil)
	if err != nil {
		panic(err)
	}

	for _, meta := range *resp.JSON200 {
		fmt.Println("Channel " + meta.Id + " was last modified at " + meta.LastModified.Format(time.RFC3339))
	}
}
