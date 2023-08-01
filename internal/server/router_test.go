package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gostuding/goMarket/internal/mocks"
)

func Test_accrualRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch r.Form.Get("id") {
		case "1":
			w.WriteHeader(http.StatusTooManyRequests)
		case "2":
			w.Write([]byte(`{"order": "2", "status": "NEW", "accrual": 0}`)) //nolint:all // <- senselessly
		case "3":
			w.Write([]byte(`{"status":  "accrual": 0}`)) //nolint:all // <- senselessly
		default:
			w.Write([]byte("")) //nolint:all // <- senselessly
		}
	}))
	defer server.Close()
	ctrl := gomock.NewController(t)
	m := mocks.NewMockCheckOrdersStorage(ctrl)
	m.EXPECT().SetOrderData("2", "NEW", float32(0)).Return(nil)

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
		{
			name: "Ограничение ожидания",
			args: args{
				url:  fmt.Sprintf("%s/?id=1", server.URL),
				strg: m,
			},
			want:    60,
			wantErr: true,
		},
		{
			name: "Без ошибок",
			args: args{
				url:  fmt.Sprintf("%s/?id=2", server.URL),
				strg: m,
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "Ошибка в json ответе",
			args: args{
				url:  fmt.Sprintf("%s/?id=3", server.URL),
				strg: m,
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			sleepChan := make(chan int, 1)
			errorChan := make(chan error, 1)
			wg := sync.WaitGroup{}
			wg.Add(1)
			go accrualRequest(tt.args.url, tt.args.strg, sleepChan, errorChan, &wg)
			wg.Wait()
			close(sleepChan)
			close(errorChan)
			if err := <-errorChan; err != nil && !tt.wantErr {
				t.Errorf("accrualRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := <-sleepChan; got != tt.want {
				t.Errorf("accrualRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
