package authenticator

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"limq/broker"
	"limq/common"
	"strings"
	"time"
)

func (a *A) GetForwardDestinations(d Descriptor) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cmd := a.c.Get(ctx, common.ForwardToDescriptor+d.Tag)
	list, err := cmd.Result()

	if err != nil {
		if !errors.Is(err, redis.Nil) {
			zap.L().Warn("redis error obtaining forward_to", zap.String("chan_id", d.Tag), zap.Error(err))
			return nil
		}
	}

	storedValues := strings.Split(list, ",")
	values := make([]string, 0, len(storedValues))

	for _, v := range storedValues {
		if len(v) == 16 {
			values = append(values, v)
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

func (a *A) CreateMixinManager() broker.MixinManager {
	return &mmanImplement{a}
}
