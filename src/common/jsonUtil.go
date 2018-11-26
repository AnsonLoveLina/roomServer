package common

import "github.com/json-iterator/go"

func JsonByte(result []byte, err error) string {
	if err != nil {
		Error.Println(err)
		return ""
	}
	return string(result)
}

func MarshalNoErrorStr(data interface{},defaultString string) string {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	result, error := json.Marshal(data)
	if error != nil {
		Error.Printf("object:%s parse error:%s", data, error)
		return defaultString
	}
	return string(result)
}

func MarshalNoError(data interface{},defaultBytes []byte) []byte {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	result, error := json.Marshal(data)
	if error != nil {
		Error.Printf("object:%s parse error:%s", data, error)
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
