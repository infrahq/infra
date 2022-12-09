package api

import "github.com/infrahq/infra/internal/validate"

const (
	DeviceFlowStatusPending   = "pending"
	DeviceFlowStatusExpired   = "expired"
	DeviceFlowStatusConfirmed = "confirmed"
)

type ApproveDeviceFlowRequest struct {
	UserCode string `json:"userCode" example:"BDSD-HQMK"`
}

func (adfr *ApproveDeviceFlowRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.String("userCode", adfr.UserCode, 8, 9, append(validate.DeviceFlowUserCode, validate.CharRange{Low: '-', High: '-'})),
	}
}

type DeviceFlowResponse struct {
	DeviceCode          string `json:"deviceCode" example:"NGU4QWFiNjQ5YmQwNG3YTdmZMEyNzQ3YzQ1YSA" note:"a code that a device will use to exchange for an access key after device login is approved"`
	VerificationURI     string `json:"verificationURI" example:"https://infrahq.com/device" note:"This is the URL the user needs to enter into their browser to start logging in"`
	UserCode            string `json:"userCode" example:"BDSD-HQMK" note:"This is the text the user will enter at the Verification URI"`
	ExpiresInSeconds    int16  `json:"expiresIn" example:"1800" note:"The number of seconds that this set of values is valid"`
	PollIntervalSeconds int8   `json:"interval" example:"5" note:"the number of seconds the device should wait between polling to see if the user has finished logging in"`
}

type DeviceFlowStatusRequest struct {
	DeviceCode string `json:"deviceCode"`
}

func (pdfr *DeviceFlowStatusRequest) ValidationRules() []validate.ValidationRule {
	return []validate.ValidationRule{
		validate.String("deviceCode", pdfr.DeviceCode, 38, 38, validate.AlphaNumeric),
	}
}

type DeviceFlowStatusResponse struct {
	Status        string         `json:"status,omitempty" note:"can be one of pending, expired, confirmed"`
	DeviceCode    string         `json:"deviceCode,omitempty" example:""`
	LoginResponse *LoginResponse `json:"login,omitempty"`
}
