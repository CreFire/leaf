package conf

import "time"

const (
	RECONNECT_INTERVAL = 1 * time.Second // 服务器间互联的客户端断线重连的时间间隔
)
