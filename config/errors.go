package config

import (
	"fmt"

	"github.com/tmzt/config-api/util"
)

type ErrReferenceNotFound struct {
	Scope         util.ScopeKind      `json:"scope"`
	AccountId     util.AccountId      `json:"account_id"`
	UserId        *util.UserId        `json:"user_id"`
	ReferenceKind ConfigReferenceKind `json:"reference_kind"`
}

func NewReferenceNotFound(scope util.ScopeKind, accountId util.AccountId, userId *util.UserId, kind ConfigReferenceKind) *ErrReferenceNotFound {
	return &ErrReferenceNotFound{
		Scope:         scope,
		AccountId:     accountId,
		UserId:        userId,
		ReferenceKind: kind,
	}
}

func (e *ErrReferenceNotFound) Error() string {
	userId := e.UserId
	if e.Scope != util.ScopeKindUser {
		userId = nil
	}
	return fmt.Sprintf("config reference not found: scope=%s accountId=%s userId=%s refKind=%s", e.Scope, e.AccountId, util.DebugStr(util.UserIdStrOrNil(userId)), e.ReferenceKind)
}

type ErrMissingRequiredParameter struct {
	ParamName string `json:"param_name"`
}

func NewMissingRequiredParameter(name string) *ErrMissingRequiredParameter {
	return &ErrMissingRequiredParameter{name}
}

func (e *ErrMissingRequiredParameter) Error() string {
	return fmt.Sprintf("missing required parameter: %s", e.ParamName)
}

type ErrInvalidConfigDataObjectHandle struct {
	Err error `json:"error"`
}

func NewInvalidConfigDataObjectHandle(err error) *ErrInvalidConfigDataObjectHandle {
	return &ErrInvalidConfigDataObjectHandle{Err: err}
}

func (e ErrInvalidConfigDataObjectHandle) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("invalid config setting handle: %v", e.Err)
	}
	return "invalid config setting handle"
}

type ErrInvalidConfigRecordType struct {
	Err error `json:"error"`
}

func NewInvalidConfigRecordType(err error) *ErrInvalidConfigRecordType {
	return &ErrInvalidConfigRecordType{Err: err}
}

func (e ErrInvalidConfigRecordType) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("invalid config record type: %v", e.Err)
	}
	return "invalid config record type"
}

func (e ErrInvalidConfigRecordType) Unwrap() error {
	return e.Err
}

type ErrInvalidConfigSettingHandle struct {
	Err error `json:"error"`
}

func NewInvalidConfigSettingHandle(err error) *ErrInvalidConfigSettingHandle {
	return &ErrInvalidConfigSettingHandle{Err: err}
}

func (e ErrInvalidConfigSettingHandle) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("invalid config setting handle: %v", e.Err)
	}
	return "invalid config setting handle"
}

type ErrConfigSettingHandleConsumed struct{}

func NewConfigSettingHandleConsumed() *ErrConfigSettingHandleConsumed {
	return &ErrConfigSettingHandleConsumed{}
}

func (e ErrConfigSettingHandleConsumed) Error() string {
	return "config setting handle already consumed"
}

type ErrConfigDataObjectEncodingFailed struct {
	Err error `json:"error"`
}

func (e *ErrConfigDataObjectEncodingFailed) Error() string {
	return fmt.Sprintf("failed to encode config data object: %v", e.Err)
}

func (e *ErrConfigDataObjectEncodingFailed) Unwrap() error {
	return e.Err
}

func NewConfigObjectEncodingFailed(err error) *ErrConfigDataObjectEncodingFailed {
	return &ErrConfigDataObjectEncodingFailed{Err: err}
}

// ErrConfigSettingError is returned when an error occurs while setting a config object or value
type ErrConfigSettingError struct {
	Err error `json:"error"`
}

func NewConfigSettingError(err error) *ErrConfigSettingError {
	return &ErrConfigSettingError{Err: err}
}

func (e *ErrConfigSettingError) Error() string {
	return fmt.Sprintf("error setting config object: %v", e.Err)
}

func (e *ErrConfigSettingError) Unwrap() error {
	return e.Err
}

// ErrConfigObjectSettingConflict is returned when a conflict is detected
type ErrConfigObjectSettingConflict struct {
	Err error `json:"error"`
}

func (e *ErrConfigObjectSettingConflict) Error() string {
	return fmt.Sprintf("conflict setting config object: %v", e.Err)
}

func (e *ErrConfigObjectSettingConflict) Unwrap() error {
	return e.Err
}

// NewConfigObjectSettingConflict returns a new and error indicating a conflict
func NewConfigObjectSettingConflict(err error) *ErrConfigObjectSettingConflict {
	return &ErrConfigObjectSettingConflict{Err: err}
}

// ErrInvalidConfigDiffParams is returned when invalid parameters are passed to the diff function
type ErrInvalidConfigDiffParams struct {
	Err error `json:"error"`
}

func NewInvalidConfigDiffParams(err error) *ErrInvalidConfigDiffParams {
	return &ErrInvalidConfigDiffParams{Err: err}
}

func (e *ErrInvalidConfigDiffParams) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("invalid diff parameters: %v", e.Err)
	}
	return "invalid diff parameters"
}

func (e *ErrInvalidConfigDiffParams) Unwrap() error {
	return e.Err
}
