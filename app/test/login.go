package test

import (
	"bytes"
	"fmt"
	"github.com/almighty/almighty-core/app"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/goatest"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// AuthorizeLoginOK test setup
func AuthorizeLoginOK(t *testing.T, ctrl app.LoginController) *app.AuthToken {
	return AuthorizeLoginOKCtx(t, context.Background(), ctrl)
}

// AuthorizeLoginOKCtx test setup
func AuthorizeLoginOKCtx(t *testing.T, ctx context.Context, ctrl app.LoginController) *app.AuthToken {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/login/authorize"), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, nil)
	authorizeCtx, err := app.NewAuthorizeLoginContext(goaCtx, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}
	err = ctrl.Authorize(authorizeCtx)
	if err != nil {
		t.Fatalf("controller returned %s, logs:\n%s", err, logBuf.String())
	}

	a, ok := resp.(*app.AuthToken)
	if !ok {
		t.Errorf("invalid response media: got %+v, expected instance of app.AuthToken", resp)
	}

	if rw.Code != 200 {
		t.Errorf("invalid response status code: got %+v, expected 200", rw.Code)
	}

	err = a.Validate()
	if err != nil {
		t.Errorf("invalid response payload: got %v", err)
	}
	return a

}

// AuthorizeLoginUnauthorized test setup
func AuthorizeLoginUnauthorized(t *testing.T, ctrl app.LoginController) {
	AuthorizeLoginUnauthorizedCtx(t, context.Background(), ctrl)
}

// AuthorizeLoginUnauthorizedCtx test setup
func AuthorizeLoginUnauthorizedCtx(t *testing.T, ctx context.Context, ctrl app.LoginController) {
	var logBuf bytes.Buffer
	var resp interface{}
	respSetter := func(r interface{}) { resp = r }
	service := goatest.Service(&logBuf, respSetter)
	rw := httptest.NewRecorder()
	req, err := http.NewRequest("GET", fmt.Sprintf("/api/login/authorize"), nil)
	if err != nil {
		panic("invalid test " + err.Error()) // bug
	}
	goaCtx := goa.NewContext(goa.WithAction(ctx, "LoginTest"), rw, req, nil)
	authorizeCtx, err := app.NewAuthorizeLoginContext(goaCtx, service)
	if err != nil {
		panic("invalid test data " + err.Error()) // bug
	}
	err = ctrl.Authorize(authorizeCtx)
	if err != nil {
		t.Fatalf("controller returned %s, logs:\n%s", err, logBuf.String())
	}

	if rw.Code != 401 {
		t.Errorf("invalid response status code: got %+v, expected 401", rw.Code)
	}

}
