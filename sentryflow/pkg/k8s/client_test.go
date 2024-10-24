// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Authors of SentryFlow

package k8s

import (
	"fmt"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestNewClient(t *testing.T) {
	type args struct {
		scheme     *runtime.Scheme
		kubeConfig string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with invalid kubeconfig path should return error",
			args: args{
				scheme:     runtime.NewScheme(),
				kubeConfig: "invalid-kubeconfig.yaml",
			},
			wantErr: true,
		},
		{
			name: "with valid kubeconfig path should return no error",
			args: args{
				scheme:     runtime.NewScheme(),
				kubeConfig: fmt.Sprintf("%s/.kube/config", os.Getenv("HOME")),
			},
			wantErr: false,
		},
		{
			name: "with empty kubeconfig path should use default config and return no error",
			args: args{
				scheme:     runtime.NewScheme(),
				kubeConfig: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClient(tt.args.scheme, tt.args.kubeConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
