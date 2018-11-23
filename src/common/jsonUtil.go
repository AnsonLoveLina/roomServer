package common


func JsonByte(result []byte,err error)string{
	if err != nil {
		Error.Println(err)
		return ""
	}
	return string(result)
}