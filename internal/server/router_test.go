package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gostuding/goMarket/internal/mocks"
)

func Test_accrualRequest(t *testing.T) {

	// TODO заменить на аргументы через GET, ?args=... r.Form.Get("id")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch string(body) {
		case "1":
			w.WriteHeader(http.StatusTooManyRequests)
		case "2":
			w.WriteHeader(http.StatusOK)
		case "3":
			w.Write([]byte(`{"status":  "accrual": 0}`))
		}
	}))
	defer server.Close()

	ctrl := gomock.NewController(t)
	m := mocks.NewMockStorage(ctrl)
	m.EXPECT().SetOrderData("uid", "1", 0).Return(nil)

	type args struct {
		url  string
		strg CheckOrdersStorage
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := accrualRequest(tt.args.url, tt.args.strg)
			if (err != nil) != tt.wantErr {
				t.Errorf("accrualRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("accrualRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
