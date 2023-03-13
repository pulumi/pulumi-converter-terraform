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
	"os"

	tfconvert "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tf2pulumi/convert"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tf2pulumi/il"
	"github.com/pulumi/pulumi/pkg/v3/codegen/convert"
	"github.com/pulumi/pulumi/sdk/v3/go/common/diag"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/cmdutil"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/contract"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/rpcutil"
	pulumirpc "github.com/pulumi/pulumi/sdk/v3/proto/go"
	"github.com/spf13/afero"
	"google.golang.org/grpc"
)

type tfConverter struct {
	pulumirpc.UnimplementedConverterServer
}

func (*tfConverter) ConvertState(ctx context.Context,
	req *pulumirpc.ConvertStateRequest,
) (*pulumirpc.ConvertStateResponse, error) {
	return nil, errors.New("not implemented")
}

func (*tfConverter) ConvertProgram(ctx context.Context,
	req *pulumirpc.ConvertProgramRequest,
) (*pulumirpc.ConvertProgramResponse, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	sink := diag.DefaultSink(os.Stderr, os.Stderr, diag.FormatOptions{
		Color: cmdutil.GetGlobalColorization(),
	})
	pluginCtx, err := plugin.NewContext(sink, sink, nil, nil, cwd, nil, true, nil)
	if err != nil {
		return nil, fmt.Errorf("create plugin host: %w", err)
	}
	defer contract.IgnoreClose(pluginCtx.Host)
	mapper, err := convert.NewPluginMapper(pluginCtx.Host, "terraform", nil)
	if err != nil {
		return nil, fmt.Errorf("create provider mapper: %w", err)
	}
	providerInfoSource := il.NewMapperProviderInfoSource(mapper)

	fs := afero.NewOsFs()
	src := afero.NewBasePathFs(fs, req.SourceDirectory)
	dst := afero.NewBasePathFs(fs, req.TargetDirectory)

	diags := tfconvert.ConvertModule(src, dst, providerInfoSource)
	if diags != nil {
		return nil, fmt.Errorf("eject program: %w", diags)
	}

	return &pulumirpc.ConvertProgramResponse{}, nil
}

func main() {
	// Fire up a gRPC server, letting the kernel choose a free port for us.
	handle, err := rpcutil.ServeWithOptions(rpcutil.ServeOptions{
		Init: func(srv *grpc.Server) error {
			pulumirpc.RegisterConverterServer(srv, &tfConverter{})
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
