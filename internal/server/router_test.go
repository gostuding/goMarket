package server

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gostuding/goMarket/internal/mocks"
)

func Test_accrualRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mocks.NewMockStorage(ctrl)
	m.EXPECT().SetOrderData("123", "NEW", 500).Return(nil)

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
