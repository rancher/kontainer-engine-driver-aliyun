package main

import (
	"encoding/json"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"strings"
)

func NewCsAPIRequest(apiName, method string) *requests.CommonRequest {
	requests.NewCommonRequest()
	request := requests.NewCommonRequest()
	request.Version = "2015-12-15"
	request.ApiName = apiName
	//request.PathPattern = "/clusters/[ClusterId]/certs"
	//request.PathParams["ClusterId"] = state.ClusterID
	request.Method = method
	request.SetDomain("cs.aliyuncs.com")
	request.SetScheme(requests.HTTPS)
	request.SetContentType("application/json;charset=utf-8")
	return request
}

func ProcessRequest(svc *cs.Client, request *requests.CommonRequest, obj interface{}) error {
	response, err := svc.ProcessCommonRequest(request)
	if err != nil && !strings.Contains(err.Error(), "JsonUnmarshalError") {
		return err
	}
	if obj != nil {
		if err := json.Unmarshal(response.GetHttpContentBytes(), obj); err != nil {
			return err
		}
	}
	return nil
}
