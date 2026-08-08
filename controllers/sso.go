package controllers

import (
	"fmt"
	"strings"
)

func (my *MainController) GetSSO() {
	if my.Ctx.Input.Method() == "GET" {
		wwwAuth := my.Ctx.Input.Header("WWW-Authenticate")
		fmt.Println(wwwAuth)
		s := strings.SplitN(my.Ctx.Input.Header("Authorization"), " ", 2)
		if len(s) != 2 || s[0] != "Negotiate" {
			//ошибка не тот запрос
			my.Ctx.ResponseWriter.Header().Set("WWW-Authenticate", "Kerberos realm=GLOBAL.BCS")
			my.Ctx.ResponseWriter.WriteHeader(401)
			my.Ctx.ResponseWriter.Write([]byte("401 Unauthorized\n"))
			return
		}
		fmt.Println("Token: ", s[1])
		return
	}

	//b, err := base64.StdEncoding.DecodeString(s[1])
	//if err != nil {
	//    return
	//}
	my.Ctx.ResponseWriter.Write([]byte{})
}
