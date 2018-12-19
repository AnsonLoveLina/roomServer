package common

import "github.com/sirupsen/logrus"

func Interface2string(o interface{}, defaultString string) string {
	if o == nil {
		return ""
	}
	result, err := o.(string)
	if err {
		logrus.WithFields(logrus.Fields{"source": o, "defaultString": defaultString}).Error("source can not transform to string,with the defaultResult")
		return defaultString
	} else {
		return result
	}
}
