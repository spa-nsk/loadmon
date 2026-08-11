package vspheremetrics

import (
	"strings"
)

func VmfsStrLunToUID(str string) string {
	size := len(str)
	if size < 32 {
		return ""
	}
	return strings.ToUpper(str[0:2] + ":" + str[2:4] + ":" + str[4:6] + ":" + str[6:8] + ":" + str[8:10] + ":" + str[10:12] + ":" + str[12:14] + ":" + str[14:16] + ":" +
		str[16:18] + ":" + str[18:20] + ":" + str[20:22] + ":" + str[22:24] + ":" + str[24:26] + ":" + str[26:28] + ":" + str[28:30] + ":" + str[30:32])
}
