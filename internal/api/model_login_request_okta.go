package api

// LoginRequestOkta struct for LoginRequestOkta
type LoginRequestOkta struct {
	Domain string `json:"domain" validate:"fqdn,required"`
	Code   string `json:"code" validate:"required"`
}
