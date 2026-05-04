package util

import (
	"errors"

	"github.com/redis/go-redis/v9"
)

func InitRedisClient() (rc *redis.Client, err error) {
	redisURL, err := MustGetString("REDIS_URL")
	if err != nil {
		return nil, errors.New("REDIS_URL missing")
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, errors.New("Failed to parse Redis URL: " + err.Error())

	}
	rc = redis.NewClient(opt)

	return rc, nil
}
