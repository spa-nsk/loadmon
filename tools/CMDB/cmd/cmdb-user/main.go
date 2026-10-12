// cmdb-user — управление локальными учётными записями из командной строки (первый администратор,
// сброс пароля). Пароль читается из переменной окружения CMDB_USER_PASSWORD или запрашивается с
// терминала без эха — в аргументах командной строки его быть не должно (виден в списке процессов).
//
//	CMDB_USER_PASSWORD=... go run ./cmd/cmdb-user add admin --role admin --name "Администратор"
//	go run ./cmd/cmdb-user passwd admin
//	go run ./cmd/cmdb-user role admin viewer
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"cmdb/internal/platform/config"
	"cmdb/internal/platform/logging"
	"cmdb/internal/repo"
	"cmdb/internal/services/auth"
	"cmdb/internal/services/svcerr"
)

func main() {
	config.ApplyEnvOverrides()
	logging.Setup("warn")
	if len(os.Args) < 3 {
		usage()
	}
	if err := repo.Init(config.LoadDBConfig()); err != nil {
		fatal(err)
	}
	ctx := context.Background()
	svc := &auth.Service{}
	cmd, name := os.Args[1], os.Args[2]

	switch cmd {
	case "add":
		fs := flag.NewFlagSet("add", flag.ExitOnError)
		role := fs.String("role", "", "роль (viewer, editor, class_owner, import_admin, auditor, accounting_admin, admin)")
		class := fs.String("class", "", "класс КЕ (для class_owner)")
		display := fs.String("name", "", "отображаемое имя")
		email := fs.String("email", "", "почта")
		_ = fs.Parse(os.Args[3:])
		id, err := svc.CreateLocal(ctx, 0, name, *display, *email, password())
		if err != nil {
			fatal(err)
		}
		if *role != "" {
			if err := auth.GrantRole(ctx, 0, id, *role, *class); err != nil {
				fatal(err)
			}
		}
		fmt.Printf("пользователь %s создан (id=%d)\n", name, id)
	case "passwd":
		id := userID(ctx, name)
		if err := svc.SetPassword(ctx, 0, id, password()); err != nil {
			fatal(err)
		}
		fmt.Printf("пароль %s изменён, сессии закрыты\n", name)
	case "role":
		if len(os.Args) < 4 {
			usage()
		}
		id := userID(ctx, name)
		class := ""
		if len(os.Args) > 4 {
			class = os.Args[4]
		}
		if err := auth.GrantRole(ctx, 0, id, os.Args[3], class); err != nil {
			fatal(err)
		}
		fmt.Printf("роль %s назначена %s\n", os.Args[3], name)
	default:
		usage()
	}
}

func userID(ctx context.Context, name string) int64 {
	users, err := auth.ListUsers(ctx)
	if err != nil {
		fatal(err)
	}
	for _, u := range users {
		if strings.EqualFold(u.Username, name) {
			var id int64
			fmt.Sscan(u.ID, &id)
			return id
		}
	}
	fatal(fmt.Errorf("%w: пользователь %q", svcerr.ErrNotFound, name))
	return 0
}

func password() string {
	if p := os.Getenv("CMDB_USER_PASSWORD"); p != "" {
		return p
	}
	fmt.Fprint(os.Stderr, "Пароль: ")
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fatal(fmt.Errorf("пароль не задан: укажите CMDB_USER_PASSWORD или запустите в терминале (%w)", err))
	}
	return string(b)
}

func usage() {
	fmt.Fprintln(os.Stderr, "использование: cmdb-user add <имя> [--role R] [--class C] [--name ФИО] [--email E] | passwd <имя> | role <имя> <роль> [класс]")
	os.Exit(2)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ошибка:", err)
	os.Exit(1)
}
