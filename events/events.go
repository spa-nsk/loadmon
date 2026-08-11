// Package events содержит простые примитивы межпоточной сигнализации
// (события) приложения, в частности канал для координированного
// завершения работы HTTP-сервера.
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
