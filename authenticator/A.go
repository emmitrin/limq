package authenticator

import "github.com/go-redis/redis/v8"

type A struct {
	c *redis.Client
}

func NewA(client *redis.Client) *A {
	return &A{client}
}
