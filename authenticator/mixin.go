package authenticator

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"limq/channel"
	"limq/common"
	"time"
)

func (a *A) GetForwardDestinations(d Descriptor) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cmd := a.c.LRange(ctx, common.ForwardToDescriptor+d.Tag, 0, -1)
	values, err := cmd.Result()

	if err != nil {
		if !errors.Is(err, redis.Nil) {
			zap.L().Warn("redis error obtaining forward_to", zap.String("chan_id", d.Tag), zap.Error(err))
			return nil
		}
	}

	return values
}

type mmanImplement struct {
	a *A
}

func (m *mmanImplement) GetForwards(tag string) []string {
	return m.a.GetForwardDestinations(Descriptor{Tag: tag})
}

func (a *A) CreateMixinManager() channel.MixinManager {
	return &mmanImplement{a}
}
