package http

import (
	"bytes"
	"github.com/v2pro/plz/countlog"
	"io/ioutil"
	"net/http"
	"unsafe"
	"github.com/v2pro/plz.service/service"
)

type Client struct {
	http.Client
	Unmarshaller service.Unmarshaller
	Marshaller   service.Marshaller
}

func NewClient() *Client {
	return &Client{
		Unmarshaller: &httpClientUnmarshaller{&jsoniterResponseUnmarshaller{}},
		Marshaller:   &httpClientMarshaller{&jsoniterMarshaller{}},
	}
}

func (client *Client) Handle(method string, url string, ptrHandlerObj interface{}) {
	ptrHandler, handlerTypeInfo := service.ConvertPtrHandler(ptrHandlerObj)
	*ptrHandler = func(ctx *countlog.Context, ptrReq unsafe.Pointer) (unsafe.Pointer, error) {
		reqObj := handlerTypeInfo.RequestBoxer(ptrReq)
		httpReq, err := http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
		err = client.Marshaller.Marshal(ctx, httpReq, reqObj)
		if err != nil {
			return nil, err
		}
		httpResp, err := client.Do(httpReq)
		if err != nil {
			return nil, err
		}
		var resp unsafe.Pointer
		ptrResp := unsafe.Pointer(&resp)
		respObj := handlerTypeInfo.ResponseBoxer(ptrResp)
		err = client.Unmarshaller.Unmarshal(ctx, respObj, httpResp)
		if err != nil {
			return nil, err
		}
		return ptrResp, nil
	}
}

type httpClientMarshaller struct {
	reqMarshaller service.Marshaller
}

func (marshaller *httpClientMarshaller) Marshal(ctx *countlog.Context, output interface{}, obj interface{}) error {
	var buf []byte
	err := marshaller.reqMarshaller.Marshal(ctx, &buf, obj)
	if err != nil {
		return err
	}
	httpReq := output.(*http.Request)
	httpReq.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
	return nil
}

type httpClientUnmarshaller struct {
	respUnmarshaller service.Unmarshaller
}

func (unmarshaller *httpClientUnmarshaller) Unmarshal(ctx *countlog.Context, obj interface{}, input interface{}) error {
	respBody, err := ioutil.ReadAll(input.(*http.Response).Body)
	if err != nil {
		return err
	}
	resp := service.Response{
		Object: obj,
	}
	err = unmarshaller.respUnmarshaller.Unmarshal(ctx, &resp, respBody)
	if err != nil {
		return err
	}
	return nil
}
