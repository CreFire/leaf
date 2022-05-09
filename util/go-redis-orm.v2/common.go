package go_redis_orm

import "errors"

type IClient interface {
	Do(commandName string, args ...interface{}) (reply interface{}, err error)
}

type NewRedisType func(dbName string, addrs []string, password string, dbindex int) (IClient, error)

func SetNewRedisHandler(handler NewRedisType) {
	gRedisMgr.SetNewRedisHandler(handler)
}

func CreateDB(dbName string, addrs []string, password string, dbindex int) error {
	return gRedisMgr.Create(dbName, addrs, password, dbindex)
}

func GetDB(dbName string) IClient {
	return gRedisMgr.Get(dbName)
}

var ERR_ISNOT_EXIST_KEY = errors.New("is not exist this key")
