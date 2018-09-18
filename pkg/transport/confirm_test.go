package transport

import (
	"net/http"
	"reflect"
	"testing"

	pb "github.com/myopenfactory/client/api"
)

func TestPrintLogs(t *testing.T) {
	type args struct {
		logs []*pb.Log
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "NullInput",
		},
		{
			name: "NullError",
			args: args{
				logs: []*pb.Log{
					{},
				},
			},
		},
		{
			name: "PrintAll",
			args: args{
				logs: []*pb.Log{
					{
						Level:       pb.Log_ERROR,
						Description: "ERROR",
					},
					{
						Level:       pb.Log_WARN,
						Description: "WARN",
					},
					{
						Level:       pb.Log_INFO,
						Description: "INFO",
					},
					{
						Level:       pb.Log_DEBUG,
						Description: "DEBUG",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrintLogs(tt.args.logs)
		})
	}
}

func TestAddLog(t *testing.T) {
	type args struct {
		logs  []*pb.Log
		Level pb.Log_Level
		msg   string
		args  []interface{}
	}
	tests := []struct {
		name string
		args args
		want []*pb.Log
	}{
		{
			name: "NilInput",
		},
		{
			name: "PlainString",
			args: args{
				Level: pb.Log_ERROR,
				msg:   "Testus",
			},
			want: []*pb.Log{
				{
					Level:       pb.Log_ERROR,
					Description: "Testus",
				},
			},
		},
		{
			name: "PrintfString",
			args: args{
				Level: pb.Log_ERROR,
				msg:   "Testus %d",
				args:  []interface{}{47},
			},
			want: []*pb.Log{
				{
					Level:       pb.Log_ERROR,
					Description: "Testus 47",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddLog(tt.args.logs, tt.args.Level, tt.args.msg, tt.args.args...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddLog() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateConfirm(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		processID string
		status    int32
		text      string
		params    []interface{}
		want      *pb.Confirm
		wantErr   bool
	}{
		{
			name:    "EmptyID",
			wantErr: true,
		},
		{
			name:    "EmptyProcessID",
			id:      "TestIdent",
			wantErr: true,
		},
		{
			name:      "EmptyText",
			id:        "TestIdent",
			processID: "4711",
			wantErr:   true,
		},
		{
			name:      "Conflict",
			id:        "TestIdent",
			processID: "4711",
			status:    http.StatusConflict,
			text:      "Test message",
			want: &pb.Confirm{
				Id:        "TestIdent",
				ProcessId: "4711",
				Logs: []*pb.Log{
					{
						Level:       pb.Log_ERROR,
						Description: "Test message",
					},
				},
				Success:    false,
				StatusCode: http.StatusConflict,
			},
			wantErr: false,
		},
		{
			name:      "Success",
			id:        "TestIdent",
			processID: "4711",
			status:    http.StatusOK,
			text:      "Test message",
			want: &pb.Confirm{
				Id:        "TestIdent",
				ProcessId: "4711",
				Logs: []*pb.Log{
					{
						Level:       pb.Log_INFO,
						Description: "Test message",
					},
				},
				Success:    true,
				StatusCode: http.StatusOK,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateConfirm(tt.id, tt.processID, tt.status, tt.text, tt.params...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateConfirm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateConfirm() = %v, want %v", got, tt.want)
			}
		})
	}
}
