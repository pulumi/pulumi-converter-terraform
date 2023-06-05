// Copyright 2016-2023, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	tfconvert "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tf2pulumi/convert"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tf2pulumi/il"
	"github.com/pulumi/pulumi/pkg/v3/codegen/convert"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/rpcutil"
	pulumirpc "github.com/pulumi/pulumi/sdk/v3/proto/go"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
)

type tfConverter struct {
}

func (*tfConverter) Close() error {
	return nil
}

func (*tfConverter) ConvertState(ctx context.Context,
	req *plugin.ConvertStateRequest,
) (*plugin.ConvertStateResponse, error) {
	return nil, errors.New("not implemented")
}

func (*tfConverter) ConvertProgram(ctx context.Context,
	req *plugin.ConvertProgramRequest,
) (*plugin.ConvertProgramResponse, error) {
	mapper, err := convert.NewMapperClient(req.MapperAddress)
	if err != nil {
		return nil, fmt.Errorf("create mapper: %w", err)
	}
	providerInfoSource := il.NewMapperProviderInfoSource(mapper)

	fs := afero.NewOsFs()
	dst := afero.NewBasePathFs(fs, req.TargetDirectory)

	diags := tfconvert.TranslateModule(fs, req.SourceDirectory, dst, providerInfoSource)
	return &plugin.ConvertProgramResponse{
		Diagnostics: diags,
	}, nil
}

func main() {
	// Fire up a gRPC server, letting the kernel choose a free port for us.
	handle, err := rpcutil.ServeWithOptions(rpcutil.ServeOptions{
		Init: func(srv *grpc.Server) error {
			pulumirpc.RegisterConverterServer(srv, plugin.NewConverterServer(&tfConverter{}))
			return nil
		},
		Options: rpcutil.OpenTracingServerInterceptorOptions(nil),
	})
	if err != nil {
		log.Fatalf("fatal: %v", err)
	}

	// The converter protocol requires that we now write out the port we have chosen to listen on.
	fmt.Printf("%d\n", handle.Port)

	// Finally, wait for the server to stop serving.
	if err := <-handle.Done; err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
