package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/cobra"
)

type DatabaseEnv struct {
	Type     string
	Addr     string
	Password string
	IsEmpty  bool
}

type DataTemplate struct {
	Type       string
	KeyCount   int
	KeySize    int
	FieldCount int
	FieldSize  int
	ValueSize  int
}

const (
	DataTypeString = "string"
	DataTypeHash   = "hash"
	DataTypeList   = "list"
	DataTypeSet    = "set"
	DataTypeZSet   = "zset"
)

var (
	dbEnv        DatabaseEnv
	dataTemplate DataTemplate
)

func newRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     dbEnv.Addr,
		Password: dbEnv.Password,
	})
}

// 生成随机字符串
func randomString(size int) string {
	var sb strings.Builder
	for i := 0; i < size; i++ {
		sb.WriteByte(byte(rand.Intn(26) + 97))
	}
	return sb.String()
}

// 根命令
var rootCmd = &cobra.Command{
	Use:   "resp-cost",
	Short: "Redis数据生成工具",
}

// 生成唯一字符串（添加在 randomString 函数附近）
func getUniqueString(num int, length int) string {
	const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	// 处理边界情况
	if num < 0 {
		num = -num
	}
	if length <= 0 {
		return ""
	}

	// 转换数字为base62
	var result []byte
	for num > 0 && len(result) < length {
		remainder := num % 62
		result = append([]byte{base62[remainder]}, result...)
		num = num / 62
	}

	// 前补零到指定长度
	for len(result) < length {
		result = append([]byte{'0'}, result...)
	}

	// 保持长度一致性
	return string(result[:length])
}

func minLengthForUniqueness(num int) int {
	if num == 0 {
		return 1
	}

	// 计算公式：62^(n-1) <= num < 62^n
	n := 1
	maxValue := 62
	for {
		if num < maxValue {
			return n
		}
		maxValue *= 62
		n++
		// 防止溢出
		if maxValue > int(^uint(0)>>1) {
			return -1 // 超出系统最大整数范围
		}
	}
}

// 检查是否可保证唯一性
func isLengthSufficient(num, length int) bool {
	return length >= minLengthForUniqueness(num)
}

func populateData(ctx context.Context, rdb *redis.Client) error {
	// init vars
	keyCount := dataTemplate.KeyCount
	keySize := dataTemplate.KeySize
	fieldCount := dataTemplate.FieldCount
	fieldSize := dataTemplate.FieldSize
	valueSize := dataTemplate.ValueSize
	dataType := dataTemplate.Type

	switch dataType {
	case DataTypeString:
		for i := 0; i < keyCount; i++ {
			key := getUniqueString(i, keySize)
			value := randomString(valueSize)
			if err := rdb.Set(ctx, key, value, 0).Err(); err != nil {
				return err
			}
		}

	case DataTypeHash:
		for i := 0; i < keyCount; i++ {
			key := getUniqueString(i, keySize)
			fields := make(map[string]interface{}, fieldCount)
			for j := 0; j < fieldCount; j++ {
				fields[getUniqueString(j, fieldSize)] = getUniqueString(j, valueSize)
			}
			if err := rdb.HSet(ctx, key, fields).Err(); err != nil {
				return err
			}
		}

	case DataTypeList:
		for i := 0; i < keyCount; i++ {
			key := getUniqueString(i, keySize)
			values := make([]interface{}, valueSize)
			for j := 0; j < valueSize; j++ {
				values[j] = getUniqueString(j, valueSize)
			}
			if err := rdb.RPush(ctx, key, values...).Err(); err != nil {
				return err
			}
		}

	case DataTypeSet:
		for i := 0; i < keyCount; i++ {
			key := getUniqueString(i, keySize)
			members := make([]interface{}, valueSize)
			for j := 0; j < valueSize; j++ {
				members[j] = getUniqueString(j, valueSize)
			}
			if err := rdb.SAdd(ctx, key, members...).Err(); err != nil {
				return err
			}
		}

	case DataTypeZSet:
		for i := 0; i < keyCount; i++ {
			key := getUniqueString(i, keySize)
			members := make([]*redis.Z, valueSize)
			for j := 0; j < valueSize; j++ {
				members[j] = &redis.Z{
					Score:  float64(j),
					Member: getUniqueString(j, valueSize),
				}
			}
			if err := rdb.ZAdd(ctx, key, members...).Err(); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("不支持的数类型: %s", dataType)
	}
	return nil
}

func populateCommand(cmd *cobra.Command, args []string) error {
	if !isLengthSufficient(dataTemplate.KeyCount, dataTemplate.KeySize) {
		fmt.Printf("key-count: %d, key-size: %d, 不满足唯一性要求\n", dataTemplate.KeyCount, dataTemplate.KeySize)
	}
	if !isLengthSufficient(dataTemplate.FieldCount, dataTemplate.FieldSize) {
		fmt.Printf("field-count: %d, field-size: %d, 不满足唯一性要求\n", dataTemplate.FieldCount, dataTemplate.FieldSize)
	}

	rdb := newRedisClient()

	if dbEnv.IsEmpty {
		rdb.FlushAll(cmd.Context())
	}

	err := populateData(cmd.Context(), rdb)
	if err != nil {
		return err
	}

	for i := 0; i < dataTemplate.KeyCount; i++ {
		rdb.Type(cmd.Context(), getUniqueString(i, dataTemplate.KeySize))
	}

	info, err := rdb.Info(cmd.Context(), "memory").Result()
	if err != nil {
		return err
	}

	lines := strings.Split(info, "\r\n")
	var used, total, max uint64
	for _, line := range lines {
		if strings.HasPrefix(line, "used_memory:") {
			fmt.Sscanf(line, "used_memory:%d", &used)
		} else if strings.HasPrefix(line, "total_system_memory:") {
			fmt.Sscanf(line, "total_system_memory:%d", &total)
		} else if strings.HasPrefix(line, "maxmemory:") {
			fmt.Sscanf(line, "maxmemory:%d", &max)
		}
	}

	log.Printf("used_memory: %d, total_system_memory: %d, maxmemory: %d\n", used, total, max)

	return nil
}

// 字符串生成命令
var populateCmd = &cobra.Command{
	Use:   "populate",
	Short: "populate redis data",
	RunE:  populateCommand,
}

func main() {
	populateCmd.Flags().StringVarP(&dbEnv.Type, "db-type", "T", "redis", "数据库类型(当前仅支持redis)")
	populateCmd.Flags().StringVarP(&dbEnv.Addr, "addr", "a", "localhost:6379", "服务器地址(格式: host:port)")
	populateCmd.Flags().StringVarP(&dbEnv.Password, "password", "p", "", "认证密码")
	populateCmd.Flags().BoolVarP(&dbEnv.IsEmpty, "empty", "e", false, "清空数据库")

	populateCmd.Flags().StringVarP(&dataTemplate.Type, "type", "t", "string", "数据类型(string|hash|list|set|zset)")
	populateCmd.Flags().IntVarP(&dataTemplate.KeyCount, "key-count", "c", 1000, "生成的数据条目数")
	populateCmd.Flags().IntVarP(&dataTemplate.KeySize, "key-size", "k", 16, "键的字节大小")
	populateCmd.Flags().IntVarP(&dataTemplate.FieldCount, "field-count", "f", 5, "每个哈希的字段数（仅hash类型有效）")
	populateCmd.Flags().IntVarP(&dataTemplate.FieldSize, "field-size", "F", 16, "字段的字节大小（仅hash类型有效）")
	populateCmd.Flags().IntVarP(&dataTemplate.ValueSize, "value-size", "s", 64, "值的字节大小（string/hash类型有效）")

	rootCmd.AddCommand(populateCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
