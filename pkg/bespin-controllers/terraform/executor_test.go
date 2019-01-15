/*
Copyright 2018 Google LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package terraform

import (
	"reflect"
	"testing"
)

func TestParseStateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     string
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "Empty config",
			cfg:     "",
			want:    map[string]string{},
			wantErr: false,
		},
		{
			name: "Good config",
			cfg: `
id              = folders/1234567
create_time     = 2000-12-05T21:32:29.614Z
display_name    = name-of-the-resource
lifecycle_state = ACTIVE

name            = folders/1234567
parent          = organizations/9876543
`,
			want: map[string]string{
				"id":              "folders/1234567",
				"create_time":     "2000-12-05T21:32:29.614Z",
				"display_name":    "name-of-the-resource",
				"lifecycle_state": "ACTIVE",
				"name":            "folders/1234567",
				"parent":          "organizations/9876543",
			},
			wantErr: false,
		},
		{
			name: "Bad config",
			cfg: `
id              = folders/1234567
create_time     = 2000-12-05T21:32:29.614Z
display_name    = name-of-the-resource
lifecycle_state = ACTIVE
name            = folders/1234567
parent          = organizations/9876543
This is a bad line.
`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseStateConfig(tc.cfg)
			switch {
			case !tc.wantErr && err != nil:
				t.Errorf("parseStateConfig() got err %+v; want nil", err)
			case tc.wantErr && err == nil:
				t.Errorf("parseStateConfig() got nil; want err %+v", tc.wantErr)
			case !reflect.DeepEqual(got, tc.want):
				t.Errorf("parseStateConfig() got %+v; want %+v", got, tc.want)
			}
		})
	}
}

func TestGetFolderID(t *testing.T) {
	tests := []struct {
		name     string
		executor *Executor
		want     int64
		wantErr  bool
	}{
		{
			name: "Valid Folder ID",
			executor: &Executor{
				state: map[string]string{
					"id":     "folders/1234567",
					"others": "value",
				},
			},
			want:    1234567,
			wantErr: false,
		},
		{
			name: "Valid Folder ID: math.MaxInt64",
			executor: &Executor{
				state: map[string]string{
					"id":     "folders/9223372036854775807",
					"others": "value",
				},
			},
			want:    9223372036854775807,
			wantErr: false,
		},
		{
			name: "Invalid Folder ID #1",
			executor: &Executor{
				state: map[string]string{
					"id":     "folders/abc123",
					"others": "value",
				},
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "Invalid Folder ID #2",
			executor: &Executor{
				state: map[string]string{
					"id":     "folders-123",
					"others": "value",
				},
			},
			want:    0,
			wantErr: true,
		},
		{

			name: "No Folder ID",
			executor: &Executor{
				state: map[string]string{},
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.executor.GetFolderID()
			switch {
			case !tc.wantErr && err != nil:
				t.Errorf("TestGetFolderID() got err %+v; want nil", err)
			case tc.wantErr && err == nil:
				t.Errorf("TestGetFolderID() got nil; want err %+v", tc.wantErr)
			case got != tc.want:
				t.Errorf("TestGetFolderID() got %+v; want %+v", got, tc.want)
			}
		})
	}
}
