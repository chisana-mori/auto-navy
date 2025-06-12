package service

import (
	"context"
	"fmt" // 添加 fmt 包导入
	"log"
	"navy-ng/models/portal" // 添加 portal 包导入
	"navy-ng/pkg/redis"     // Import redis package
	"strconv"
	"strings" // 添加 strings 包导入
	"time"
)

// InitCacheService 初始化缓存服务，包括预热和启动订阅者
func InitCacheService(deviceService *DeviceService, deviceQueryService *DeviceQueryService, deviceCache *DeviceCache) {
	// 启动后台协程进行缓存预热
	go func() {
		// 等待5秒，确保其他服务已经初始化
		time.Sleep(5 * time.Second)

		// 创建上下文
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 预热缓存
		log.Println("Starting cache warmup...")
		err := deviceCache.WarmupCache(deviceService, deviceQueryService)
		if err != nil {
			log.Printf("Cache warmup failed: %v\n", err)
		} else {
			log.Println("Cache warmup completed successfully")
		}

		// 定期刷新缓存
		go periodicCacheRefresh(ctx, deviceService, deviceQueryService, deviceCache)
	}()

	// 注册设备变更事件发布器
	// 将 service 包中的 publishDeviceChangeEvent 函数传递给 models 包
	portal.RegisterDeviceChangeEventPublisher(publishDeviceChangeEvent)

	// 启动设备更新订阅者 Goroutine, 传入 deviceService 和 deviceCache
	go startDeviceUpdateSubscriber(deviceService, deviceCache)
}

// periodicCacheRefresh 定期刷新缓存
func periodicCacheRefresh(ctx context.Context, deviceService *DeviceService, deviceQueryService *DeviceQueryService, deviceCache *DeviceCache) {
	// 每小时刷新一次缓存
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Starting periodic cache refresh...")
			err := deviceCache.WarmupCache(deviceService, deviceQueryService)
			if err != nil {
				log.Printf("Periodic cache refresh failed: %v\n", err)
			} else {
				log.Println("Periodic cache refresh completed successfully")
			}

			// 打印缓存统计信息
			stats := deviceCache.GetCacheStats()
			log.Printf("Cache stats - Hits: %d, Misses: %d, Sets: %d, Deletes: %d\n",
				stats.Hits, stats.Misses, stats.Sets, stats.Deletes)

			// 每天重置一次统计信息
			if time.Now().Hour() == 0 {
				deviceCache.ResetCacheStats()
				log.Println("Cache stats reset")
			}
		case <-ctx.Done():
			log.Println("Stopping periodic cache refresh")
			return
		}
	}
}

// startDeviceUpdateSubscriber 启动一个后台 Goroutine 来监听设备更新消息并更新/失效缓存
func startDeviceUpdateSubscriber(deviceService *DeviceService, deviceCache *DeviceCache) {
	log.Println("Starting device update subscriber...")

	// 检查依赖项
	if deviceService == nil || deviceCache == nil {
		log.Println("Error: DeviceService or DeviceCache is nil in subscriber. Cannot start.")
		return
	}

	// 创建 Redis Handler
	// 确保使用与发布者相同的 Redis 实例配置
	redisHandler := redis.NewRedisHandler("default")
	if redisHandler == nil {
		log.Println("Error: Redis handler is not available for subscriber.")
		return // 无法启动订阅者
	}

	// 订阅通道
	// 注意：DeviceUpdatesChannel 需要在此处可访问，或者直接使用字符串 "navy:device:updates"
	// 为了简单起见，我们直接使用字符串。更好的做法是在公共地方定义常量。
	const deviceUpdatesChannel = "navy:device:updates"
	pubsub := redisHandler.Subscribe(deviceUpdatesChannel)
	defer pubsub.Close() // 确保在函数退出时关闭订阅

	// 检查订阅是否成功 (可选，但建议)
	_, err := pubsub.Receive(context.Background())
	if err != nil {
		log.Printf("Error subscribing to channel %s: %v\n", deviceUpdatesChannel, err)
		return
	}

	// 获取消息通道
	ch := pubsub.Channel()
	log.Printf("Successfully subscribed to channel: %s\n", deviceUpdatesChannel)

	// 循环处理消息
	for msg := range ch {
		log.Printf("Received message from channel %s: %s\n", msg.Channel, msg.Payload)

		// 解析消息（设备 ID）
		deviceIDStr := msg.Payload
		deviceID, err := strconv.ParseInt(deviceIDStr, 10, 0)
		if err != nil {
			log.Printf("Error parsing device ID from message '%s': %v\n", deviceIDStr, err)
			continue // 继续处理下一条消息
		}

		// 使用后台上下文调用 GetDevice
		ctx := context.Background()
		latestDeviceData, err := deviceService.GetDevice(ctx, int(deviceID))

		if err != nil {
			// 检查是否是 "not found" 错误 (表示设备被删除)
			// 注意：GetDevice 返回的是格式化后的错误，需要检查字符串内容
			if strings.Contains(err.Error(), fmt.Sprintf(ErrDeviceNotFoundMsg, deviceID)) {
				log.Printf("Device ID %d not found in DB (likely deleted), invalidating cache entry.\n", deviceID)
				// 失效（删除）单个设备缓存
				deviceCache.InvalidateDevice(int(deviceID))
			} else {
				// 其他错误，记录日志
				log.Printf("Error fetching device ID %d from DB: %v\n", deviceID, err)
				// 即使获取失败，也尝试失效缓存，以防数据不一致
				deviceCache.InvalidateDevice(int(deviceID))
			}
		} else {
			// 获取成功，更新缓存
			log.Printf("Updating cache for device ID: %d\n", deviceID)
			deviceCache.SetDevice(int(deviceID), latestDeviceData)
		}

		// 无论更新还是删除，都失效列表缓存（简单策略）
		log.Printf("Invalidating device list caches due to change in device ID: %d\n", deviceID)
		listErr := deviceCache.InvalidateDeviceLists()
		if listErr != nil {
			log.Printf("Error invalidating device lists cache for ID %d: %v\n", deviceID, listErr)
		}
	}

	// 如果循环退出（通常是 PubSub 关闭时），记录日志
	log.Println("Device update subscriber stopped.")
}
