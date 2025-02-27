package main

import (
	"fmt"

	"github.com/CreFire/leaf/util/goredis"
	"github.com/gomodule/redigo/redis"
)

func testStandalone() {
	option := goredis.NewDefaultOption()
	option.Type = goredis.Standalone
	option.Password = "123456"
	addrs := []string{"172.26.144.21:6379"}
	db, err0 := goredis.NewClient("", addrs, option)
	if err0 != nil {
		fmt.Println(err0)
		return
	}
	_, err1 := db.Do("SET", "a", "12345")
	if err1 != nil {
		fmt.Println(err1)
		return
	}
	a, err2 := redis.Int(db.Do("GET", "a"))
	if err2 != nil {
		fmt.Println(err2)
		return
	}
	fmt.Println("a =", a)
}
