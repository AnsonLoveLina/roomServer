package common

func Interface2string(o interface{}, defaultString string) string {
	if o == nil {
		return ""
	}
	result, err := o.(string)
	if err {
		Error.Printf("source:%s can not transform to string,use defaultResult:%s", o, defaultString)
		return defaultString
	} else {
		return result
	}
}
