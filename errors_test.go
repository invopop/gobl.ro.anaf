package roefactura_test

import (
	"errors"
	"testing"

	roefactura "github.com/invopop/gobl.ro.anaf"
)

func TestErrorMessage(t *testing.T) {
	cause := errors.New("bad gateway")

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "the key alone", err: roefactura.ErrConversion, want: "conversion"},
		{name: "a message replaces the key", err: roefactura.ErrConversion.WithMsg("cannot render %s", "xml"), want: "cannot render xml"},
		{name: "the cause follows the message", err: roefactura.ErrConversion.WithMsg("cannot render").WithCause(cause), want: "cannot render (bad gateway)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.err.Error(); got != test.want {
				t.Errorf("message: want %q, got %q", test.want, got)
			}
		})
	}
}

func TestErrorMatching(t *testing.T) {
	cause := errors.New("bad gateway")
	err := roefactura.ErrConversion.WithMsg("cannot render").WithCause(cause)

	if !errors.Is(err, roefactura.ErrConversion) {
		t.Error("an error must match its own kind")
	}
	if errors.Is(err, roefactura.ErrUnreadable) {
		t.Error("an error must not match another kind")
	}
	if !errors.Is(err, cause) {
		t.Error("an error must match the cause it wraps")
	}
	if !errors.Is(roefactura.ErrConversion.WithCause(roefactura.ErrSkipped), roefactura.ErrSkipped) {
		t.Error("an error must match a wrapped error of another kind")
	}

	var target *roefactura.Error
	if !errors.As(err, &target) {
		t.Error("errors.As must find the converter error")
	}
	if errors.Unwrap(err) != cause {
		t.Error("unwrap must return the cause")
	}
}
