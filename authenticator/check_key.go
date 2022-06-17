package authenticator

import (
	"context"
	"limq/common"
)

const (
	permRedisKey = `permissions`
	tagRedisKey  = `channel_id`
)

type Descriptor struct {
	Tag   string
	Flags AccessLevel
}

func (a *A) CheckAccessKey(key string) Descriptor {
	hash := common.ChannelDescriptor + key

	response := a.c.HGetAll(context.Background(), hash)
	result, err := response.Result()
	if err != nil {
		return Descriptor{}
	}

	d := Descriptor{Tag: result[tagRedisKey],
		Flags: parseAccessLevel(result[permRedisKey])}

	return d
}
