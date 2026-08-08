package events

var (
	shutdownServerHTTP chan bool
)

func init() {
	shutdownServerHTTP = make(chan bool)
}

//ShutdownServerHTTP - установить флаг отключения сервера HTTP
func ShutdownServerHTTP() {
	shutdownServerHTTP <- true
}

//ChanelShutdownServerHTTP - получить адрес канала
func ChanelShutdownServerHTTP() *chan bool {
	return &shutdownServerHTTP
}
