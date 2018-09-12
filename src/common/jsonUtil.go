package common

import "log"

func JsonByte(result []byte,err error)string{
	if err != nil {
		log.Println(err)
		return ""
	}
	return string(result)
}