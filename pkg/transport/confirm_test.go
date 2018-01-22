package transport

import (
	"reflect"
	"testing"

	pb "myopenfactory.io/x/api/tatooine"
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
					&pb.Log{},
				},
			},
		},
		{
			name: "PrintAll",
			args: args{
				logs: []*pb.Log{
					&pb.Log{
						Level:       pb.Log_ERROR,
						Description: "ERROR",
					},
					&pb.Log{
						Level:       pb.Log_WARN,
						Description: "WARN",
					},
					&pb.Log{
						Level:       pb.Log_INFO,
						Description: "INFO",
					},
					&pb.Log{
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
				&pb.Log{
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
				&pb.Log{
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
	type args struct {
		msg    *pb.Message
		status int32
		text   string
		params []interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.Confirm
		wantErr bool
	}{
		{
			name: "NilInputMsg",
			args: args{
				msg: &pb.Message{
					Id:        "TestIdent",
					ProcessId: "4711",
				},
			},
			wantErr: true,
		},
		{
			name: "Success",
			args: args{
				msg: &pb.Message{
					Id:        "TestIdent",
					ProcessId: "4711",
				},
				text: "Test message",
			},
			want: &pb.Confirm{
				ProcessId: "4711",
				Id:        "TestIdent",
				Success:   true,
				Logs: []*pb.Log{
					&pb.Log{
						Description: "Test message",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Success",
			args: args{
				msg: &pb.Message{
					Id:        "TestIdent",
					ProcessId: "4711",
				},
				text:   "Test message",
				status: 409,
			},
			want: &pb.Confirm{
				ProcessId: "4711",
				Id:        "TestIdent",
				Success:   false,
				Logs: []*pb.Log{
					&pb.Log{
						Description: "Test message",
						Level:       pb.Log_ERROR,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateConfirm(tt.args.msg.Id, tt.args.msg.ProcessId, tt.args.status, tt.args.text, tt.args.params...)
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
