package conf

var (
	LenStackBuf = 4096

	// log
	PrintLevel string
	LogLevel   string
	LogPath    string
	LogPrint   bool
	LogFileOne bool
	LogFlag    int

	// console
	ConsolePort   int
	ConsolePrompt string = "Server# "
	ProfilePath   string

	// cluster
	ListenAddr      string
	ConnAddrs       []string
	PendingWriteNum int
)
