// Package grpcclient — gRPC-клиент пакета loadmon. Обращается к серверу
// пакета grpcserver, который отдаёт метрики VMware vSphere (сервис Loadmon,
// адрес config.GetCliApiAddr()) и данные из БД (сервис LoadmonDB, адрес
// config.GetCliDBAddr()), а также к серверу HPE OneView за инвентарём
// оборудования.
//
// Общий стиль функций пакета:
//   - на каждый вызов открывается короткоживущее соединение через dialAPI()/
//     dialDB()/dialAPIAddr() с таймаутом 30 секунд на весь запрос (включая
//     чтение потоковых ответов), соединение закрывается через defer сразу
//     после установления;
//   - при ошибке подключения или вызова RPC в лог пишется причина
//     (log.Printf), а вызывающему возвращается "пустой" результат (nil/nil-
//     срез/false) и, где это предусмотрено сигнатурой, сама ошибка;
//   - при потоковом чтении (stream.Recv()) признаком конца потока служит
//     io.EOF, любая другая ошибка прерывает или пропускает элемент в
//     зависимости от функции (см. комментарии на местах).
package grpcclient

import (
	"context"
	"loadmon/config"
	pb "loadmon/protocol"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// dialTimeout — таймаут на установление соединения и выполнение одного
// gRPC-запроса (включая чтение потокового ответа целиком).
const dialTimeout = 30 * time.Second

// StatusWork описывает результат серверной операции (например, очистки
// метрик): State — код состояния (0 — успех, иное — ошибка/предупреждение),
// Details — человекочитаемое описание результата.
type StatusWork struct {
	State   int64
	Details string
}

// dialAPIAddr устанавливает соединение с gRPC-сервисом Loadmon (API метрик
// vSphere/OneView) по произвольному адресу addr и возвращает готового
// клиента, соединение, контекст с таймаутом dialTimeout и функцию его
// отмены. Вызывающая сторона обязана закрыть соединение (defer conn.Close())
// и отменить контекст (defer cancel()) сразу после получения результата.
// При ошибке подключения причина логируется и возвращается в err; conn,
// ctx и cancel в этом случае равны нулевым значениям и использовать их нельзя.
func dialAPIAddr(addr string) (pb.LoadmonClient, *grpc.ClientConn, context.Context, context.CancelFunc, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("did not connect: %v", err)
		return nil, nil, nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	return pb.NewLoadmonClient(conn), conn, ctx, cancel, nil
}

// dialAPI — сокращённая форма dialAPIAddr для адреса основного API-сервера
// loadmon, взятого из config.GetCliApiAddr().
func dialAPI() (pb.LoadmonClient, *grpc.ClientConn, context.Context, context.CancelFunc, error) {
	return dialAPIAddr(config.GetCliApiAddr())
}

// dialDB устанавливает соединение с gRPC-сервисом LoadmonDB (работа со
// справочниками/БД) по адресу config.GetCliDBAddr() и возвращает готового
// клиента, соединение, контекст с таймаутом dialTimeout и функцию его
// отмены. Правила использования и обработки ошибки такие же, как у
// dialAPIAddr.
func dialDB() (pb.LoadmonDBClient, *grpc.ClientConn, context.Context, context.CancelFunc, error) {
	conn, err := grpc.NewClient(config.GetCliDBAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("did not connect: %v", err)
		return nil, nil, nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	return pb.NewLoadmonDBClient(conn), conn, ctx, cancel, nil
}

// LoadVCenterList запрашивает у API-сервера loadmon список подключённых
// vCenter'ов, имена которых содержат filter (пустая строка — все). Использует
// адрес по умолчанию (config.GetCliApiAddr()). При ошибке подключения или RPC
// возвращает пустой (не nil) срез — ошибка логируется внутри.
