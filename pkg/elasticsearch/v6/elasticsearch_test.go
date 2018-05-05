package v6

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/olivere/elastic"
)

func Test_parseRequest(t *testing.T) {
	type args struct {
		bulkableRequests []elastic.BulkableRequest
	}

	// `{"index": {"_id": "5a892848-c3d4-4554-9bdd-abc0d2445c8f", "_index": "docker-2018.05.04", "_type": "log"}}`,
	id := uuid.New().String()
	msg := `{
		"containerID": "c84dd5a73553", "containerName": "webapper",
		"containerImageName": "rchicoli/webapper", "containerCreated": "2018-05-04T23:49:33.313835788+02:00",
		"message": "message1", "source": "stdout", "timestamp": "2018-05-04T23:52:20.71878058+02:00", "partial": false
	}`
	r := elastic.NewBulkIndexRequest().Index("test").Doc(msg).Id(id)

	bulkableRequest := make([]elastic.BulkableRequest, 0)
	bulkableRequest = append(bulkableRequest, r)

	tests := []struct {
		name    string
		args    args
		want    *mapRequests
		wantErr error
	}{
		{name: "parseRequest()", args: args{bulkableRequest}, want: &mapRequests{requests: map[string]string{id: msg}}, wantErr: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRequest(tt.args.bulkableRequests)
			if err != tt.wantErr {
				t.Errorf("parseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mapRequests_ById(t *testing.T) {
	type fields struct {
		requests map[string]string
	}
	type args struct {
		id string
	}

	id := uuid.New().String()
	msg := `{
		"containerID": "c84dd5a73553", "containerName": "webapper",
		"containerImageName": "rchicoli/webapper", "containerCreated": "2018-05-04T23:49:33.313835788+02:00",
		"message": "message1", "source": "stdout", "timestamp": "2018-05-04T23:52:20.71878058+02:00", "partial": false
	}`

	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{name: "request mapped", fields: fields{requests: map[string]string{id: msg}}, args: args{id: id}, want: msg},
		{name: "request not found", fields: fields{requests: map[string]string{"test": msg}}, args: args{id: id}, want: "request not found"},
		{name: "request is nil", fields: fields{requests: nil}, args: args{id: id}, want: "request not found"},
		{name: "get request by id with empty string", fields: fields{requests: nil}, args: args{id: ""}, want: "request not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &mapRequests{
				requests: tt.fields.requests,
			}
			if got := p.ById(tt.args.id); got != tt.want {
				t.Errorf("mapRequests.ById() = %v, want %v", got, tt.want)
			}
		})
	}
}
