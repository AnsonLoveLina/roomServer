package common

import (
	"github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

func JsonByte(result []byte, err error) string {
	if err != nil {
		logrus.Error(err)
		return ""
	}
	return string(result)
}

func MarshalNoErrorStr(data interface{},defaultString string) string {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	result, error := json.Marshal(data)
	if error != nil {
		logrus.WithFields(logrus.Fields{"object": data, "error": error}).Error("json marshal parse error")
		return defaultString
	}
	return string(result)
}

func MarshalNoError(data interface{},defaultBytes []byte) []byte {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	result, error := json.Marshal(data)
	if error != nil {
		logrus.WithFields(logrus.Fields{"object": data, "error": error}).Error("json marshal parse error")
		return defaultBytes
	}
	return result
}

func Marshal(data interface{}) ([]byte, error) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Marshal(data)
}

func UnMarshal(input []byte, data *interface{}) error {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Unmarshal(input, data)
}
