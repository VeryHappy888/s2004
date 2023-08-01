package db

import (
	"errors"
	"fmt"
	"github.com/gogf/gf/encoding/gjson"
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/util/gconv"
	"github.com/gomodule/redigo/redis"
	"strconv"
	"strings"
	"time"
)

const (
	REDIS_OPERATION_SET    = "SET"
	REDIS_OPERATION_GET    = "GET"
	REDIS_OPERATION_EXISTS = "EXISTS"
	REDIS_OPERATION_DELETE = "DEL"
)

// NewRedisPool 返回redis连接池
func NewRedisPool() *redis.Pool {
	conf := g.Cfg().GetString("redis.default")
	v := strings.Split(conf, ":")
	cf := strings.Split(v[1], ",")
	ps := ""
	if len(cf) > 2 {
		ps = cf[2]
	}
	d, _ := strconv.Atoi(cf[1])
	return &redis.Pool{
		MaxActive:   0,
		MaxIdle:     0,
		IdleTimeout: 5000 * time.Second,
		Dial: func() (redis.Conn, error) {
			db := redis.DialDatabase(d)
			pass := redis.DialPassword(ps)
			c, err := redis.DialURL("redis://"+v[0], db, pass)
			if err != nil {
				return nil, fmt.Errorf("redis connection error: %s", err)
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			if err != nil {
				return fmt.Errorf("ping redis error: %s", err)
			}
			return nil
		},
	}
}

/*
*
定义推送结构
*/
type PushMsg struct {
	UserName uint64
	Type     int
	Time     int64
	Data     interface{}
	Text     string
}

/***
* 发送队例
 */
func PushQueue(i interface{}) {
	queueName := g.Cfg().GetString("redis.topic")
	if i == nil {
		return
	}
	con := NewRedisPool().Get()
	defer con.Close()
	_, err := con.Do("rpush", queueName, gconv.String(i))
	if err != nil {
		fmt.Println("redis连接异常", err.Error())
	}
}

func SETExpirationObj(k string, i interface{}, expiration int64) error {
	redisConn := NewRedisPool().Get()
	defer redisConn.Close()
	iData, err := gjson.Encode(i)
	if err != nil {
		return err
	}
	var result interface{}
	if expiration > 0 {
		result, err = redisConn.Do(REDIS_OPERATION_SET, k, iData, "EX", expiration)
	} else {
		result, err = redisConn.Do(REDIS_OPERATION_SET, k, iData)
	}

	if err != nil {
		//logger.Errorln(err)
		return err
	}
	if gconv.String(result) == "OK" {
		return nil
	}
	return errors.New(gconv.String(result))
}

func SETObj(k string, i interface{}) error {
	redisConn := NewRedisPool().Get()
	defer redisConn.Close()
	iData, err := gjson.Encode(i)
	if err != nil {
		return err
	}
	result, err := redisConn.Do(REDIS_OPERATION_SET, k, iData)
	if err != nil {
		//logger.Errorln(err)
		return err
	}
	if gconv.String(result) == "OK" {
		return nil
	}
	return errors.New(gconv.String(result))
}

func GETObj(k string, i interface{}) error {
	redisConn := NewRedisPool().Get()
	defer redisConn.Close()
	_var, err := redisConn.Do(REDIS_OPERATION_GET, k)
	if err != nil {
		return err
	}
	err = gjson.DecodeTo(_var, &i)
	if err != nil {
		return err
	}
	return nil
}

func DelObj(k string) error {
	redisConn := NewRedisPool().Get()
	defer redisConn.Close()
	_, err := redisConn.Do(REDIS_OPERATION_DELETE, k)
	if err != nil {
		return err
	}
	return nil
}

func Exists(k string) (bool, error) {
	//检查是否存在key值
	redisConn := NewRedisPool().Get()
	defer redisConn.Close()
	exists, err := redisConn.Do(REDIS_OPERATION_EXISTS, k)
	if err != nil {
		fmt.Println("illegal exception")
		return false, err
	}
	//fmt.Printf("exists or not: %v \n", exists)
	if exists.(int64) == 1 {
		return true, nil
	}
	return false, nil
}
