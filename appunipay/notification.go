package appunipay

// appstore server notify v2
type AppstoreServerNotifyV2 struct {
	SignedPayload string `json:"signedPayload"`
}

// AppstoreDecodedPayload
// doc: https://developer.apple.com/documentation/appstoreservernotifications/responsebodyv2decodedpayload
type AppstoreDecodedPayload struct {
	// https://developer.apple.com/documentation/appstoreservernotifications/notificationtype
	NotificationType string `json:"notificationType"`

	SubType             string                  `json:"subtype"`
	NotificationUUID    string                  `json:"notificationUUID"`
	NotificationVersion string                  `json:"notificationVersion"`
	Data                NotificationPayloadData `json:"data"`
}

// doc: https://developer.apple.com/documentation/appstoreservernotifications/data
type NotificationPayloadData struct {
	AppAppleId            int64          `json:"appAppleId"`
	BundleId              string         `json:"bundleId"`
	BundleVersion         string         `json:"bundleVersion"`
	Environment           string         `json:"environment"`
	SignedRenewalInfo     JWSRenewalInfo `json:"signedRenewalInfo"`
	SignedTransactionInfo JWSTransaction `json:"signedTransactionInfo"`
}

type JWSRenewalInfo string
type JWSTransaction string

type JWSDecodedHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	X5c string `json:"x5c"`
}
