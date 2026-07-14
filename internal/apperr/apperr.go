package apperr

import (
	"errors"
	"net/http"
)

type Detail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Target  string `json:"target"`
}

type AppError struct {
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	Details  []Detail `json:"details,omitempty"`
	HTTPCode int     `json:"-"`
}

func (e *AppError) Error() string {
	return e.Code + ": " + e.Message
}

func New(code, message string, httpCode int) *AppError {
	return &AppError{Code: code, Message: message, HTTPCode: httpCode}
}

func WithDetails(code, message string, httpCode int, details []Detail) *AppError {
	return &AppError{Code: code, Message: message, Details: details, HTTPCode: httpCode}
}

func As(err error) (*AppError, bool) {
	var target *AppError
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

var (
	Validation = New("VALIDATION_ERROR", "请求参数校验失败", http.StatusOK)

	InvalidCredentials = New("INVALID_CREDENTIALS", "用户名或密码错误", http.StatusUnauthorized)
	WrongPassword      = New("WRONG_PASSWORD", "旧密码错误", http.StatusOK)
	SamePassword       = New("SAME_PASSWORD", "新密码与旧密码相同", http.StatusOK)
	WeakPassword       = New("WEAK_PASSWORD", "密码不符合强度规则", http.StatusOK)

	Unauthorized = New("UNAUTHORIZED", "未认证或 Token 无效", http.StatusUnauthorized)
	InvalidToken  = New("INVALID_TOKEN", "refresh token 无效或已过期", http.StatusUnauthorized)
	Forbidden     = New("FORBIDDEN", "权限不足", http.StatusForbidden)

	CannotDeleteBuiltin = New("CANNOT_DELETE_BUILTIN", "不能删除内置用户", http.StatusOK)
	CannotDemoteBuiltin = New("CANNOT_DEMOTE_BUILTIN", "不能降级内置用户", http.StatusOK)

	UserNotFound   = New("USER_NOT_FOUND", "用户不存在", http.StatusOK)
	UserExists     = New("USER_EXISTS", "用户已存在", http.StatusOK)
	ZoneNotFound   = New("ZONE_NOT_FOUND", "Zone 不存在", http.StatusOK)
	ZoneExists     = New("ZONE_EXISTS", "Zone 已存在", http.StatusOK)
	DomainNotFound = New("DOMAIN_NOT_FOUND", "Domain 不存在", http.StatusOK)
	DomainExists   = New("DOMAIN_EXISTS", "Domain 已存在", http.StatusOK)
	DomainZoneConflict = New("DOMAIN_ZONE_CONFLICT", "Domain 全名或其祖先名命中既有 Zone", http.StatusOK)
	RecordNotFound     = New("RECORD_NOT_FOUND", "Record 不存在", http.StatusOK)
	RecordExists       = New("RECORD_EXISTS", "同 Domain 下已存在重复记录", http.StatusOK)
	RecordTypeInvalid  = New("RECORD_TYPE_INVALID", "Record 类型非法", http.StatusOK)
	RecordIdExhausted  = New("RECORD_ID_EXHAUSTED", "Domain 下 record-id 序号达上限", http.StatusOK)

	ServiceUnavailable = New("SERVICE_UNAVAILABLE", "etcd 服务不可用", http.StatusServiceUnavailable)
	InternalError      = New("INTERNAL_ERROR", "服务器内部错误", http.StatusInternalServerError)
)

func ValidationError(details []Detail) *AppError {
	return WithDetails("VALIDATION_ERROR", "请求参数校验失败", http.StatusOK, details)
}
