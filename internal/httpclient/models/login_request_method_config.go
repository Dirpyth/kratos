// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/swag"
)

// LoginRequestMethodConfig LoginRequestMethodConfig login request method config
// swagger:model loginRequestMethodConfig
type LoginRequestMethodConfig struct {
	LoginRequestMethodConfigAllOf0

	LoginRequestMethodConfigAllOf1

	LoginRequestMethodConfigAllOf2

	LoginRequestMethodConfigAllOf3
}

// UnmarshalJSON unmarshals this object from a JSON structure
func (m *LoginRequestMethodConfig) UnmarshalJSON(raw []byte) error {
	// AO0
	var aO0 LoginRequestMethodConfigAllOf0
	if err := swag.ReadJSON(raw, &aO0); err != nil {
		return err
	}
	m.LoginRequestMethodConfigAllOf0 = aO0

	// AO1
	var aO1 LoginRequestMethodConfigAllOf1
	if err := swag.ReadJSON(raw, &aO1); err != nil {
		return err
	}
	m.LoginRequestMethodConfigAllOf1 = aO1

	// AO2
	var aO2 LoginRequestMethodConfigAllOf2
	if err := swag.ReadJSON(raw, &aO2); err != nil {
		return err
	}
	m.LoginRequestMethodConfigAllOf2 = aO2

	// AO3
	var aO3 LoginRequestMethodConfigAllOf3
	if err := swag.ReadJSON(raw, &aO3); err != nil {
		return err
	}
	m.LoginRequestMethodConfigAllOf3 = aO3

	return nil
}

// MarshalJSON marshals this object to a JSON structure
func (m LoginRequestMethodConfig) MarshalJSON() ([]byte, error) {
	_parts := make([][]byte, 0, 4)

	aO0, err := swag.WriteJSON(m.LoginRequestMethodConfigAllOf0)
	if err != nil {
		return nil, err
	}
	_parts = append(_parts, aO0)

	aO1, err := swag.WriteJSON(m.LoginRequestMethodConfigAllOf1)
	if err != nil {
		return nil, err
	}
	_parts = append(_parts, aO1)

	aO2, err := swag.WriteJSON(m.LoginRequestMethodConfigAllOf2)
	if err != nil {
		return nil, err
	}
	_parts = append(_parts, aO2)

	aO3, err := swag.WriteJSON(m.LoginRequestMethodConfigAllOf3)
	if err != nil {
		return nil, err
	}
	_parts = append(_parts, aO3)

	return swag.ConcatJSON(_parts...), nil
}

// Validate validates this login request method config
func (m *LoginRequestMethodConfig) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *LoginRequestMethodConfig) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *LoginRequestMethodConfig) UnmarshalBinary(b []byte) error {
	var res LoginRequestMethodConfig
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}

// LoginRequestMethodConfigAllOf0 login request method config all of0
// swagger:model LoginRequestMethodConfigAllOf0
type LoginRequestMethodConfigAllOf0 interface{}

// LoginRequestMethodConfigAllOf1 login request method config all of1
// swagger:model LoginRequestMethodConfigAllOf1
type LoginRequestMethodConfigAllOf1 interface{}

// LoginRequestMethodConfigAllOf2 login request method config all of2
// swagger:model LoginRequestMethodConfigAllOf2
type LoginRequestMethodConfigAllOf2 interface{}

// LoginRequestMethodConfigAllOf3 login request method config all of3
// swagger:model LoginRequestMethodConfigAllOf3
type LoginRequestMethodConfigAllOf3 interface{}
