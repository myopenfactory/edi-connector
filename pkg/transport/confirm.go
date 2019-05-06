package transport

import (
	"fmt"
	"net/http"

	pb "github.com/myopenfactory/client/api"
	"github.com/myopenfactory/client/pkg/log"
)

// CreateConfirm create a new pb.Confirm object and add one log entry
func CreateConfirm(id, processid string, status int32, text string, params ...interface{}) (*pb.Confirm, error) {
	if id == "" {
		return nil, fmt.Errorf("error id required")
	}
	if processid == "" {
		return nil, fmt.Errorf("error processid required")
	}
	if text == "" {
		return nil, fmt.Errorf("error text is required")
	}
	success := true
	lvl := pb.Log_INFO
	if status != http.StatusOK {
		success = false
		lvl = pb.Log_ERROR
	}

	return &pb.Confirm{
		Id:         id,
		ProcessId:  processid,
		Logs:       AddLog([]*pb.Log{}, lvl, text, params...),
		Success:    success,
		StatusCode: status,
	}, nil
}

// AddLog appends a log entry to the log list
func AddLog(logs []*pb.Log, Level pb.Log_Level, msg string, args ...interface{}) []*pb.Log {
	if len(msg) == 0 {
		return logs
	}
	return append(logs, &pb.Log{
		Description: fmt.Sprintf(msg, args...),
		Level:       Level,
	})
}

// PrintLogs prints all log entries to logging framework
func PrintLogs(logger *log.Logger, logs []*pb.Log) {
	for _, logentry := range logs {
		switch logentry.Level {
		case pb.Log_ERROR:
			logger.Errorf("%s", logentry.Description)
		case pb.Log_WARN:
			logger.Warnf("%s", logentry.Description)
		case pb.Log_INFO:
			logger.Infof("%s", logentry.Description)
		case pb.Log_DEBUG:
			logger.Debugf("%s", logentry.Description)
		}
	}
}
