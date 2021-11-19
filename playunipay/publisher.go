package playunipay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/api/androidpublisher/v3"
)

type AndroidPublisherApis struct {
	VerifyProduct      string
	AckProduct         string
	VerifySubscription string
	AckSubscription    string
	CancelSubscription string
	RefundSubscription string
	RevokeSubscription string
}

// RemoteAndroidPublisherService
// 解决国内无法访问google服务的问题
type RemoteAndroidPublisherService struct {
	Client *http.Client
	Apis   AndroidPublisherApis

	Endpoint string
}

var RemoteAndroidPublisherApis = AndroidPublisherApis{
	VerifyProduct:      "/google/iap/verifyProduct",
	AckProduct:         "/google/iap/ackProduct",
	VerifySubscription: "/google/iap/verifySubscription",
	AckSubscription:    "/google/iap/ackSubscription",
	CancelSubscription: "/google/iap/cancelSubscription",
	RefundSubscription: "/google/iap/refundSubscription",
	RevokeSubscription: "/google/iap/revokeSubscription",
}

func ErrorResponse(resp *http.Response) error {
	data := make(map[string]interface{})

	err := json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return err
	}

	msg := "Error:"
	for k, v := range data {
		item := fmt.Sprintf("%s:%v", k, v)
		msg += item
	}

	return errors.New(msg)
}

func (svc RemoteAndroidPublisherService) Do(req *http.Request, result interface{}) error {
	resp, err := svc.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ErrorResponse(resp)
	}

	err = json.NewDecoder(resp.Body).Decode(result)

	return err
}

func (svc RemoteAndroidPublisherService) VerifySubscription(ctx context.Context, packageName string, subscriptionId string, token string) (*androidpublisher.SubscriptionPurchase, error) {
	body := map[string]interface{}{
		"packageName":    packageName,
		"subscriptionID": subscriptionId,
		"purchaseToken":  token,
	}

	buf, _ := json.Marshal(body)
	req, err := http.NewRequest("POST",
		svc.Endpoint+svc.Apis.VerifySubscription, bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	var data androidpublisher.SubscriptionPurchase
	err = svc.Do(req, &data)

	return &data, err
}

func (svc RemoteAndroidPublisherService) AcknowledgeSubscription(ctx context.Context, packageName string, subscriptionId string, token string, req *androidpublisher.SubscriptionPurchasesAcknowledgeRequest) error {
	body := map[string]interface{}{
		"packageName":    packageName,
		"subscriptionID": subscriptionId,
		"purchaseToken":  token,
	}
	if req.DeveloperPayload != "" {
		body["developerPayload"] = req.DeveloperPayload
	}

	if len(req.ForceSendFields) > 0 {
		body["forceSendFields"] = req.ForceSendFields
	}

	if len(req.NullFields) > 0 {
		body["nullFields"] = req.NullFields
	}

	buf, _ := json.Marshal(body)
	httpreq, err := http.NewRequest("POST", svc.Endpoint+svc.Apis.AckSubscription, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	var data struct{}
	return svc.Do(httpreq, &data)
}

func (svc RemoteAndroidPublisherService) CancelSubscription(ctx context.Context, packageName string, subscriptionId string, token string) error {
	body := map[string]interface{}{
		"packageName":    packageName,
		"subscriptionID": subscriptionId,
		"purchaseToken":  token,
	}

	buf, _ := json.Marshal(body)
	req, err := http.NewRequest("POST",
		svc.Endpoint+svc.Apis.CancelSubscription, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	var data struct{}
	return svc.Do(req, &data)
}

func (svc RemoteAndroidPublisherService) RefundSubscription(ctx context.Context, packageName string, subscriptionId string, token string) error {
	body := map[string]interface{}{
		"packageName":    packageName,
		"subscriptionID": subscriptionId,
		"purchaseToken":  token,
	}

	buf, _ := json.Marshal(body)
	req, err := http.NewRequest("POST",
		svc.Endpoint+svc.Apis.RefundSubscription, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	var data struct{}
	return svc.Do(req, &data)
}

func (svc RemoteAndroidPublisherService) RevokeSubscription(ctx context.Context, packageName string, subscriptionId string, token string) error {
	body := map[string]interface{}{
		"packageName":    packageName,
		"subscriptionID": subscriptionId,
		"purchaseToken":  token,
	}

	buf, _ := json.Marshal(body)
	req, err := http.NewRequest("POST",
		svc.Endpoint+svc.Apis.RevokeSubscription, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	var data struct{}
	return svc.Do(req, &data)
}

func (svc RemoteAndroidPublisherService) VerifyProduct(ctx context.Context, packageName string, subscriptionId string, token string) (*androidpublisher.ProductPurchase, error) {
	body := map[string]interface{}{
		"packageName":    packageName,
		"subscriptionID": subscriptionId,
		"purchaseToken":  token,
	}

	buf, _ := json.Marshal(body)
	req, err := http.NewRequest("POST",
		svc.Endpoint+svc.Apis.VerifyProduct, bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}

	var data androidpublisher.ProductPurchase
	err = svc.Do(req, &data)
	return &data, err
}

func (svc RemoteAndroidPublisherService) AcknowledgeProduct(ctx context.Context, packageName string, subscriptionId string, token, developerPayload string) error {
	body := map[string]interface{}{
		"packageName":      packageName,
		"subscriptionID":   subscriptionId,
		"purchaseToken":    token,
		"developerPayload": developerPayload,
	}

	buf, _ := json.Marshal(body)
	req, err := http.NewRequest("POST",
		svc.Endpoint+svc.Apis.AckProduct, bytes.NewBuffer(buf))
	if err != nil {
		return err
	}

	var data struct{}
	return svc.Do(req, &data)
}
