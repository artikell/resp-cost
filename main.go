package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"

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
	shardValue   []sync.Map
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

// 生成唯一字符串
func getUniqueString(num int, length int) string {
	if len(shardValue) <= length {
		newShardValue := make([]sync.Map, length+1)
		copy(newShardValue, shardValue)
		shardValue = newShardValue
	}

	shard := shardValue[length]
	if _, ok := shard.Load(num); !ok {
		format := "%" + strconv.Itoa(length) + "d"
		shard.Store(num, fmt.Sprintf(format, num))
	}
	v, _ := shard.Load(num)
	return v.(string)
}

func minLengthForUniqueness(num int) int {
	return len(strconv.Itoa(num))
}

// 检查是否可保证唯一性
func isLengthSufficient(num, length int) bool {
	return length >= minLengthForUniqueness(num)
}

func populateData(ctx context.Context, rdb *redis.Client, dt *DataTemplate, waitGroup *sync.WaitGroup, ds int, de int) error {
	// init vars
	keySize := dt.KeySize
	fieldCount := dt.FieldCount
	fieldSize := dt.FieldSize
	valueSize := dt.ValueSize
	dataType := dt.Type

	switch dataType {
	case DataTypeString:
		for i := ds; i < de; i++ {
			key := getUniqueString(i, keySize)
			value := getUniqueString(i, valueSize)
			if err := rdb.Set(ctx, key, value, 0).Err(); err != nil {
				return err
			}
		}
	case DataTypeHash:
		for i := ds; i < de; i++ {
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
		for i := ds; i < de; i++ {
			key := getUniqueString(i, keySize)
			values := make([]interface{}, fieldCount)
			for j := 0; j < fieldCount; j++ {
				values[j] = getUniqueString(j, fieldSize)
			}
			if err := rdb.RPush(ctx, key, values...).Err(); err != nil {
				return err
			}
		}

	case DataTypeSet:
		for i := ds; i < de; i++ {
			key := getUniqueString(i, keySize)
			members := make([]interface{}, fieldCount)
			for j := 0; j < fieldCount; j++ {
				members[j] = getUniqueString(j, fieldSize)
			}
			if err := rdb.SAdd(ctx, key, members...).Err(); err != nil {
				return err
			}
		}

	case DataTypeZSet:
		for i := ds; i < de; i++ {
			key := getUniqueString(i, keySize)
			members := make([]*redis.Z, fieldCount)
			for j := 0; j < fieldCount; j++ {
				members[j] = &redis.Z{
					Score:  float64(j),
					Member: getUniqueString(j, fieldSize),
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

func flushDatabase(ctx context.Context, rdb *redis.Client) error {
	err := rdb.FlushAll(ctx).Err()
	if err != nil {
		return err
	}
	var usedBefore, usedAfter uint64
	for {
		// 获取flush后的内存使用情况
		infoAfter, err := rdb.Info(ctx, "memory").Result()
		if err != nil {
			return err
		}
		for _, line := range strings.Split(infoAfter, "\r\n") {
			if strings.HasPrefix(line, "used_memory:") {
				fmt.Sscanf(line, "used_memory:%d", &usedAfter)
				break
			}
		}
		if usedBefore*100 > usedAfter*5 {
			usedBefore = usedAfter
			continue
		}
		break
	}
	return nil
}

func populateCommand(cmd *cobra.Command, args []string) error {
	if !isLengthSufficient(dataTemplate.KeyCount, dataTemplate.KeySize) {
		return fmt.Errorf("key-count: %d, key-size: %d, 不满足唯一性要求\n", dataTemplate.KeyCount, dataTemplate.KeySize)
	}
	if (dataTemplate.Type != DataTypeString) &&
		!isLengthSufficient(dataTemplate.FieldCount, dataTemplate.FieldSize) {
		return fmt.Errorf("field-count: %d, field-size: %d, 不满足唯一性要求\n", dataTemplate.FieldCount, dataTemplate.FieldSize)
	}

	rdb := newRedisClient()

	if dbEnv.IsEmpty {
		flushDatabase(cmd.Context(), rdb)
	}

	wg := sync.WaitGroup{}
	goroutineCount := 12
	keyPreRound := dataTemplate.KeyCount / goroutineCount
	for i := 0; i < dataTemplate.KeyCount; {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ds := i
			de := i + keyPreRound
			err := populateData(cmd.Context(), rdb, &dataTemplate, &wg, ds, de)
			if err != nil {
				panic(err)
			}
		}(i)
		i = i + keyPreRound
	}
	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ds := i * dataTemplate.KeyCount / goroutineCount
			de := (i + 1) * dataTemplate.KeyCount / goroutineCount
			err := populateData(cmd.Context(), rdb, &dataTemplate, &wg, ds, de)
			if err != nil {
				panic(err)
			}
		}(i)
	}
	wg.Wait()

	// 使用 SCAN 命令遍历所有 key
	var cursor uint64 = 0
	for {
		var err error
		_, cursor, err = rdb.Scan(cmd.Context(), cursor, "*", int64(dataTemplate.KeyCount)).Result()
		if err != nil {
			return err
		}

		if cursor == 0 {
			break
		}
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

	fmt.Printf("used_memory: %d, total_system_memory: %d, maxmemory: %d\n", used, total, max)

	return nil
}

// 字符串生成命令
var populateCmd = &cobra.Command{
	Use:   "populate",
	Short: "populate redis data",
	RunE:  populateCommand,
}

func main() {
	shardValue = make([]sync.Map, 10)

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
