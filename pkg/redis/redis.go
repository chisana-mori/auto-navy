package redis

import (
	"strings"
	"time"

	"github.com/go-redis/redis"
)

type _client struct {
	cli *redis.Client
}

var clientMap map[string]_client

func init() {
	clientMap = make(map[string]_client)
	Init("default", "127.0.0.1:6379", "")
}

func Init(dbName string, host string, password string) error {
	hostSlice := strings.Split(host, ",")
	client := redis.NewClient(&redis.Options{
		Addr:     hostSlice[0],
		Password: password,
		DB:       0,
	})
	_, err := client.Ping().Result()
	if err != nil {
		return err
	}
	clientMap[dbName] = _client{cli: client}
	return nil
}

type Handler struct {
	client            *redis.Client
	DefaultExpiration time.Duration
}

func NewRedisHandler(db string) *Handler {
	handler := &Handler{DefaultExpiration: time.Hour * 24}
	handler.client = Client(db)
	return handler
}

func Client(db string) *redis.Client {
	return clientMap[db].cli
}

// Keys 使用 Redis KEYS 命令查找匹配的键。
// 警告：此方法在生产环境中可能导致性能问题，因为它会阻塞服务器。
// 推荐使用 ScanKeys 方法代替。
func (rh *Handler) Keys(pattern string) (keys []string, err error) {
	keys, err = rh.client.Keys(pattern).Result()
	return
}

// ScanKeys 使用 Redis SCAN 命令迭代查找匹配的键，避免阻塞服务器。
func (rh *Handler) ScanKeys(pattern string) ([]string, error) {
	var cursor uint64
	var keys []string
	var err error

	for {
		var currentKeys []string
		// 执行 SCAN 命令，每次扫描默认数量的键（通常是 10）
		currentKeys, cursor, err = rh.client.Scan(cursor, pattern, 10).Result()
		if err != nil {
			return nil, err // 返回错误
		}

		keys = append(keys, currentKeys...) // 将本次扫描到的键追加到结果列表

		// 如果游标返回 0，表示迭代完成
		if cursor == 0 {
			break
		}
	}

	return keys, nil // 返回所有找到的键
}

func (rh *Handler) Expire(expiration time.Duration) {
	rh.DefaultExpiration = expiration
}

func (rh *Handler) Set(key string, value interface{}) {
	err := rh.client.Set(key, value, rh.DefaultExpiration).Err()
	if err != nil {
	}
}

func (rh *Handler) SetWithExpireTime(key string, value string, expiry time.Duration) {
	err := rh.client.Set(key, value, expiry).Err()
	if err != nil {
	}
}

func (rh *Handler) AcquireLock(key string, value string, expiry time.Duration) (isSuccess bool, err error) {
	isSuccess, err = rh.client.SetNX(key, value, expiry).Result()
	if err != nil {
		return // Explicit return on error might be intended here
	}
	return // Returns the isSuccess and err values
}

func (rh *Handler) Pub(channel string, message string) (err error) {
	err = rh.client.Publish(channel, message).Err()
	if err != nil {
		return
	}
	return
}

func (rh *Handler) Subscribe(channel string) (ret *redis.PubSub) {
	ret = rh.client.Subscribe(channel)
	return
}

func (rh *Handler) Delete(key string) {
	rh.client.Del(key)
}

func (rh *Handler) Exist(key string) (flag bool) {
	count, err := rh.client.Exists(key).Result()
	if err != nil {
	}
	if count != 0 {
		flag = true
	}
	return
}

func (rh *Handler) Get(key string) string {
	val, err := rh.client.Get(key).Result()
	if err != nil {
		return ""
	}
	return val
}
